package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const TheQ = "http://thequestion.ru/questions/next/%d"

type URL *url.URL

func main() {
	var proxyFilePtr = flag.String(
		"proxy-list", "",
		"File with a list of proxies to use")
	flag.Parse()

	proxyList, err := getProxyList(*proxyFilePtr)
	if err != nil {
		log.Panicf(
			"Can't read list of proxies form '%s': %v.",
			*proxyFilePtr, err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyList[0]),
		}}

	var pageUrl = fmt.Sprintf(TheQ, 55)
	q, err := fetchQuestion(pageUrl, httpClient)
	if err != nil {
		log.Panic(err)
	}
	log.Println(*q)
	for _, ans := range (*q).Answers {
		log.Println(*ans)
	}
}

func getProxyList(fileName string) ([]URL, error) {
	if fileName == "" {
		// Proxy list is not provided, return "fake proxy" with URL=nil
		// to connect directly.
		return []URL{nil}, nil
	}

	txt, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var lines = strings.Split(string(txt), "\n")
	var proxies = make([]URL, len(lines))
	for i, ln := range lines {
		proxyUrl, err := url.Parse(ln)
		if err != nil {
			return nil, err
		}
		proxies[i] = proxyUrl
	}
	return proxies, nil
}

type Question struct {
	Id      int
	Json    string
	Answers []*Answer
}

type Answer struct {
	Id     uint64
	Json   string
	UserId uint64
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
	q, err := parseQuestion(body)
	if err != nil {
		return nil, err
	}
	return q, nil
}

// get answer id form <div id=answer-1234>
var AnswerIdRx = regexp.MustCompile("^answer-(\\d+)$")

// get user id from account URL
var UserIdRx = regexp.MustCompile("^/account/(\\d+)")

func parseQuestion(body []byte) (*Question, error) {
	var rawQ map[string]interface{}
	if err := json.Unmarshal(body, &rawQ); err != nil {
		return nil, err
	}

	var qId = rawQ["questionId"]
	var qHtml = rawQ["questionHTML"]
	if qHtml == nil || qId == nil {
		return nil, fmt.Errorf("Unexepcted JSON document")
	}

	var htmlReader = strings.NewReader(rawQ["questionHTML"].(string))
	var htmlTokenizer = html.NewTokenizer(htmlReader)
	var question = Question{Id: int(qId.(float64))}
	var answer = &Answer{}

tokenLoop:
	for {
		switch htmlTokenizer.Next() {
		case html.ErrorToken:
			break tokenLoop // end of document

		case html.StartTagToken:
			tok := htmlTokenizer.Token()
			if attr := getAttr("div", "question-data", tok); attr != nil {
				question.Json = (*attr).Val
			} else if attr := getAttr("div", "id", tok); attr != nil {
				if match := AnswerIdRx.FindStringSubmatch((*attr).Val); match != nil {
					if res, err := strconv.ParseUint(match[1], 10, 64); err == nil {
						(*answer).Id = res
					}
				}
			} else if attr := getAttr("script", "type", tok); attr != nil {
				if (*attr).Val == "application/ld+json" {
					htmlTokenizer.Next() // get text node next to <script> tag
					(*answer).Json = htmlTokenizer.Token().Data
				}
			} else if attr := getAttr("a", "class", tok); attr != nil {
				if (*attr).Val == "answer__account-username" {
					attr := getAttr("a", "href", tok)
					if match := UserIdRx.FindStringSubmatch((*attr).Val); match != nil {
						if res, err := strconv.ParseUint(match[1], 10, 64); err == nil {
							(*answer).UserId = res
							question.Answers = append(question.Answers, answer)
							answer = &Answer{}
						}
					}
				}
			}

		default:
			continue
		}
	}

	return &question, nil
}

func getAttr(tagName string, attrName string, t html.Token) *html.Attribute {
	if t.Data == tagName {
		for _, attr := range t.Attr {
			if attr.Key == attrName {
				return &attr
			}
		}
	}
	return nil
}
