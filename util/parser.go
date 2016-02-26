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
	var question = Question{Id: int(qId.(float64))}
	var answer = &Answer{}

tokenLoop:
	for {
		switch htmlTokenizer.Next() {
		case html.ErrorToken:
			break tokenLoop // end of document

    // Parser is very fragile, it expects data bits to occur in order.
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
							// All fileds in `answer` are set up, add it to the `question` and
							// allocate the new one to proceed.
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

// getAddr checks if token matches tagName and searches for attr with
// specified attrName.
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
