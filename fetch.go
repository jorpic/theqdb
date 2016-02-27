package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

import (
	"database/sql"
	_ "github.com/lib/pq"
)

import . "github.com/jorpic/theqdb/util"

const TheQ = "http://thequestion.ru/questions/next/%d"

func main() {
	config := GetConfig()

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(config.ProxyList[0])}}

	var pageUrl = fmt.Sprintf(TheQ, 155)
	q, err := fetchQuestion(pageUrl, httpClient)
	if err != nil {
		log.Panic(err)
	}

	db, err := sql.Open("postgres", config.PgConnString)
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect DB: %v", err)
	}
	defer db.Close()

	err = dbInsertQuestion(db, q)
	if err != nil {
		log.Panic(err)
	}
}

func dbInsertQuestion(db *sql.DB, q *Question) error {
	var err error = nil
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`insert into raw_question(id, data)
        values ($1::int, $2::jsonb)`,
		q.Id, q.Json)
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, ans := range q.Answers {
		_, err = tx.Exec(
			`insert into raw_answer(id, user_id, data)
          values ($1::int, $2::int, $3::jsonb)`,
			ans.Id, ans.UserId, ans.Json)
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
