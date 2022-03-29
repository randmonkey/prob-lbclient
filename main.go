package main

import (
	"log"
	"prob-lbclient/lbclient"
	"time"
)

var (
	targetURL      = "https://www.ebay.com"
	requestTimeout = time.Second
)

func main() {
	r := lbclient.NewResolver([]string{"8.8.8.8"})
	c, err := lbclient.NewClient(targetURL, 100)
	if err != nil {
		log.Fatalf("failed to create client, error %v", err)
	}
	c.WithRequestTimeout(requestTimeout)
	c.WithResolver(r)
	c.SendRequests()
}
