package util

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"regexp"
	"strconv"
	"strings"
)

// get answer id form <div id=answer-1234>
var AnswerIdRx = regexp.MustCompile("^answer-(\\d+)$")

// get user id from account URL
var UserIdRx = regexp.MustCompile("^/account/(\\d+)")

func ParseQuestion(body []byte) (*Question, error) {
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
	var question = Question{Id: uint64(qId.(float64))}
	var answer = &Answer{}

tokenLoop:
	for {
		switch htmlTokenizer.Next() {
		case html.ErrorToken:
			break tokenLoop // end of document

			// Parser is very fragile, it expects data bits to occur in order.
		case html.StartTagToken:
			var m = match(htmlTokenizer.Token())
			if m.tag("div").attr("question-data").to(&question.Json) {
				continue
			}
			if m.tag("div").attr("id").extract(AnswerIdRx).toInt(&answer.Id) {
				continue
			}
			if m.tag("script").attr("type").val("application/ld+json") {
				htmlTokenizer.Next() // get text node next to <script> tag
				answer.Json = htmlTokenizer.Token().Data
				continue
			}
			if m.tag("a").attr("class").val("answer__account-username") {
				m.tag("a").attr("href").extract(UserIdRx).toInt(&answer.UserId)
				// All fileds in `answer` are set up, add it to the `question` and
				// allocate the new one to proceed.
				question.Answers = append(question.Answers, answer)
				answer = &Answer{}
			}

		default:
			continue
		}
	}

	return &question, nil
}

// Everyting below is ugly but simple DSL
// for matching & extacting parts of HTML.
// ---------------------------------------
type Matcher struct {
	Token *html.Token
	Val   string
	Match bool
}

func match(t html.Token) *Matcher {
	return &Matcher{Token: &t, Match: true}
}

func (mp *Matcher) tag(name string) *Matcher {
	mp.Match = mp.Token.Data == name
	return mp
}

func (mp *Matcher) attr(name string) *Matcher {
	if mp.Match {
		mp.Match = false
		for _, attr := range mp.Token.Attr {
			if attr.Key == name {
				mp.Val = attr.Val
				mp.Match = true
				break
			}
		}
	}
	return mp
}

func (mp *Matcher) extract(rx *regexp.Regexp) *Matcher {
	if mp.Match {
		mp.Match = false
		if rxMatch := rx.FindStringSubmatch(mp.Val); rxMatch != nil {
			mp.Val = rxMatch[1]
			mp.Match = true
		}
	}
	return mp
}

func (mp *Matcher) to(val *string) bool {
	if mp.Match {
		*val = mp.Val
	}
	return (*mp).Match
}

func (mp *Matcher) toInt(val *uint64) bool {
	if mp.Match {
		mp.Match = false
		if res, err := strconv.ParseUint(mp.Val, 10, 64); err == nil {
			*val = res
			mp.Match = true
		}
	}
	return (*mp).Match
}

func (mp *Matcher) val(val string) bool {
	return mp.Match && val == mp.Val
}
