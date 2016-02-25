package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
)

const TheQ = "http://thequestion.ru/questions/next/%i"

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
		if proxyList == nil {
			log.Panicf(
				"Can't read list of proxies form '%s': %v.",
				*proxyFilePtr, err)
		}
	} else {
		// URL = nil means "do not use proxy"
		proxyList = []URL{nil}
	}

	log.Println(proxyList)
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

func fetchQuestion(id int) string {
	return ""
}
