package util

import (
	"flag"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
)

// Config holds various options that control program execution.
type Config struct {
	// List of proxy URLs to use.
	// URL can be `nil` to connect without proxy.
	ProxyList     []*url.URL
	PgConnString  string
	MaxQuestionId int
	Threads       int
}

// GetConfig parses config from command line options.
func GetConfig() *Config {
	config := Config{}
	flag.StringVar(
		&config.PgConnString,
		"db", "dbname=theq",
		"PG connection string to use")
	flag.IntVar(
		&config.MaxQuestionId,
		"max-id", 100,
		"Fetch all the questions up to max-id")
	flag.IntVar(
		&config.Threads,
		"threads", 8,
		"Number of threads to spawn")
	proxyFilePtr := flag.String(
		"proxy-list", "",
		"File with a list of proxies to use")
	flag.Parse()

	proxyList, err := getProxyList(*proxyFilePtr)
	if err != nil {
		log.Fatalf(
			"Can't read list of proxies form '%s': %v.",
			*proxyFilePtr, err)
	}
	config.ProxyList = proxyList
	return &config
}

func getProxyList(fileName string) ([]*url.URL, error) {
	if fileName == "" {
		// Proxy list is not provided, return "fake proxy" with URL=nil
		// to connect without proxy.
		return []*url.URL{nil}, nil
	}

	txt, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var lines = strings.Split(string(txt), "\n")
	var proxies = make([]*url.URL, len(lines))
	for i, ln := range lines {
		proxyURL, err := url.Parse(ln)
		if err != nil {
			return nil, err
		}
		proxies[i] = proxyURL
	}
	return proxies, nil
}
