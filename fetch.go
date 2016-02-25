package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const TheQ = "http://thequestion.ru/questions/next/%d"

type URL *url.URL

func main() {
	var proxyFilePtr = flag.String(
		"proxy-list", "",
		"File with a list of proxies to use")
	flag.Parse()

	var err error
	var proxyList []URL

	if *proxyFilePtr != "" {
		proxyList, err = readProxyList(*proxyFilePtr)
		if err != nil {
			log.Panicf(
				"Can't read list of proxies form '%s': %v.",
				*proxyFilePtr, err)
		}
	} else {
		// URL = nil means "do not use proxy"
		proxyList = []URL{nil}
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyList[0]),
		}}

	var pageUrl = fmt.Sprintf(TheQ, 50)
	q, err := fetchQuestion(pageUrl, httpClient)
	if err != nil {
		log.Panic(err)
	}
	log.Println((*q).q)
}

func readProxyList(fileName string) ([]URL, error) {
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
	q string
}

func fetchQuestion(url string, client *http.Client) (*Question, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	txt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	q, err := parseQuestion(string(txt))
	if err != nil {
		return nil, err
	}
	return q, nil
}

func parseQuestion(txt string) (*Question, error) {
	return &Question{q: txt}, nil
}
