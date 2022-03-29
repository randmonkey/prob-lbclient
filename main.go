package main

import (
	"flag"
	"log"
	"prob-lbclient/lbclient"
	"strings"
	"time"
)

var (
	targetURL      = "https://www.ebay.com"
	dnsResolver    = "8.8.8.8"
	requestTimeout = time.Second
	requestNumber  = 100
)

func main() {
	flag.StringVar(&targetURL, "url", targetURL, "target URL, default to "+targetURL)
	flag.StringVar(&dnsResolver, "dns", dnsResolver, "dns resolver, seperate by `,` if multiple")
	flag.DurationVar(&requestTimeout, "t", requestTimeout, "timeout for each request")
	flag.IntVar(&requestNumber, "n", requestNumber, "number of requests to send")
	flag.Parse()

	r := lbclient.NewResolver(strings.Split(dnsResolver, ","))
	c, err := lbclient.NewClient(targetURL, requestNumber)
	if err != nil {
		log.Fatalf("failed to create client, error %v", err)
	}
	c.WithRequestTimeout(requestTimeout)
	c.WithResolver(r)
	c.SendRequests()
}
