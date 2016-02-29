package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

import (
	"database/sql"
	_ "github.com/lib/pq"
)

import . "github.com/jorpic/theqdb/util"

const theQ = "http://thequestion.ru/questions/next/%d"

func main() {
	config := GetConfig()

	db, _ := sql.Open("postgres", config.PgConnString)
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect DB: %v", err)
	}
	defer db.Close()

	// supply of question IDs to fetch
	jobs := make(chan int)
	go jobGenerator(db, config.MaxQuestionId, jobs)

	// start fetching threads
	var wg sync.WaitGroup
	for i := 0; i < config.Threads; i++ {
		wg.Add(1)
		go worker(&wg, db, jobs, config)
	}
	wg.Wait()
}

func jobGenerator(db *sql.DB, maxID int, jobs chan int) {
	// get IDs of missing questions
	rows, err := db.Query(
		`select generate_series(a+1, b-1) as id
		   from
		     (select
		         lag(id, 1, 0) over (order by id asc) as a,
		         id as b
		       from raw_question) x
		   where a+1 <> b
		   union (select coalesce(max(id),0)+1 from raw_question)
		   order by id`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var id int
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			log.Fatal(err)
		}
		jobs <- id
	}
	if err := rows.Err(); err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}

	// IDs of rest of questions
	for ; id <= maxID; id++ {
		jobs <- id
	}

	close(jobs)
}

func worker(wg *sync.WaitGroup, db *sql.DB, jobSrc chan int, config *Config) {
	proxies := config.ProxyList
	var goodProxies = []*url.URL{}

	for {
		if len(proxies) == 0 {
			if len(goodProxies) == 0 {
				log.Printf("Terminating worker: proxy list is empty.")
				wg.Done()
				return
			}
			log.Printf("===== %d proxies survived round!", len(goodProxies))
			for _, p := range goodProxies {
				log.Println(p.Host)
			}
			proxies = goodProxies
			goodProxies = []*url.URL{}
		}

		for _, i := range rand.Perm(len(proxies)) {
			proxy := proxies[i]

			id, gotJob := <-jobSrc
			if !gotJob {
				wg.Done()
				return
			}

			httpClient := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxy)}}

			var pageURL = fmt.Sprintf(theQ, id)

			q, err := fetchQuestion(pageURL, httpClient)
			if err != nil {
				continue
			}
			goodProxies = append(goodProxies, proxy)

			err = dbInsertQuestion(db, q)
			if err != nil {
				log.Printf("Failed to store %d: %v", id, err)
			}
		}
	}
}

func dbInsertQuestion(db *sql.DB, q *Question) error {
	var err error
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`insert into raw_question(id, data)
			values ($1::int, $2::jsonb)`,
		q.ID, q.JSON)
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, ans := range q.Answers {
		_, err = tx.Exec(
			`insert into raw_answer(id, user_id, data)
				values ($1::int, $2::int, $3::jsonb)`,
			ans.ID, ans.UserID, ans.JSON)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

var errBadProxy = fmt.Errorf("Bad proxy?")
var errMalformedQuestion = fmt.Errorf("Malformed question")
var errRateLimit = fmt.Errorf("Rate limit")

func fetchQuestion(url string, client *http.Client) (*Question, error) {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Googlebot/2.1 (+http://www.google.com/bot.html)")
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil, errBadProxy
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	q, err := ParseQuestion(body)
	if err != nil {
		b := string(body)
		if strings.Contains(b, "enot") || strings.Contains(b, "<title>Ошибка") {
			return nil, errRateLimit
		}
		log.Println(string(body), err)
		return nil, errMalformedQuestion
	}
	return q, nil
}
