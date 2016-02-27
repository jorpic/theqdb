package util

import (
	"flag"
	"log"
	"net/url"
	"io/ioutil"
	"strings"
)


// Config holds various options that control program execution.
type Config struct {
	// List of proxy URLs to use.
	// URL can be `nil` to connect without proxy.
	ProxyList    []*url.URL
	PgConnString string
}

// GetConfig parses config from command line options.
func GetConfig() *Config {
	var proxyFilePtr = flag.String(
		"proxy-list", "",
		"File with a list of proxies to use")
	var dbConnStringPtr = flag.String(
		"db", "dbname=theq",
		"PG connection string to use")
	flag.Parse()

	proxyList, err := getProxyList(*proxyFilePtr)
	if err != nil {
		log.Fatalf(
			"Can't read list of proxies form '%s': %v.",
			*proxyFilePtr, err)
	}
  return &Config{
    ProxyList: proxyList,
    PgConnString: *dbConnStringPtr}
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
