package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
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
		`select generate_series(a+1, b-1)
		   from
		     (select
		         lag(id, 1, 0) over (order by id asc) as a,
		         id as b
		       from raw_question) x
		   where a+1 <> b
		 union (select coalesce(max(id),0)+1 from raw_question)`)
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
	if err := rows.Err(); err != sql.ErrNoRows {
		log.Fatal(err)
	}

	// IDs of rest of questions
	for ; id <= maxID; id++ {
		jobs <- id
	}

	close(jobs)
}

func worker(wg *sync.WaitGroup, db *sql.DB, jobSrc chan int, config *Config) {
	for {
		id, gotJob := <-jobSrc
		if !gotJob {
			wg.Done()
			return
		}

		proxies := config.ProxyList
		randomProxy := proxies[rand.Intn(len(proxies))]
		httpClient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(randomProxy)}}

		var pageURL = fmt.Sprintf(theQ, id)

		q, err := fetchQuestion(pageURL, httpClient)
		if err != nil {
			log.Printf("Failed to fetch %d: %v", id, err)
			continue
		}

		err = dbInsertQuestion(db, q)
		if err != nil {
			log.Printf("Failed to store %d: %v", id, err)
			continue
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

func fetchQuestion(url string, client *http.Client) (*Question, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	q, err := ParseQuestion(body)
	if err != nil {
		return nil, err
	}
	return q, nil
}
