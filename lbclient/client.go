package lbclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

type Client struct {
	URL            *url.URL
	Requests       int
	RequestTimeout time.Duration

	MaxConcurrentRequests int
	RequestInterval       time.Duration

	HandleResponse func(*http.Response)

	resolver     *Resolver
	availableIPs []string
	selector     Selector

	successRequests int32
	sendControlCh   chan struct{}
	requestResultCh chan requestResult
}

type requestResult struct {
	code        int
	err         error
	ip          string
	responseDur time.Duration
}

func NewClient(targetURL string, requestCount int) (*Client, error) {
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "http://" + targetURL
	}
	u, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	c := &Client{
		URL:      u,
		Requests: requestCount,

		MaxConcurrentRequests: 4,
		RequestInterval:       10 * time.Millisecond,

		HandleResponse: func(resp *http.Response) {
			fmt.Printf("response from %s: code %d\n", resp.Request.URL, resp.StatusCode)
		},
	}

	c.requestResultCh = make(chan requestResult, c.MaxConcurrentRequests)
	c.sendControlCh = make(chan struct{}, c.MaxConcurrentRequests)

	return c, nil
}

func (c *Client) WithRequestTimeout(dur time.Duration) *Client {
	c.RequestTimeout = dur
	return c
}

func (c *Client) WithResolver(r *Resolver) *Client {
	c.resolver = r
	return c
}

// set interval of each request.
func (c *Client) WithRequestInterval(interval time.Duration) *Client {
	c.RequestInterval = interval
	return c
}

// set max concurrent requests that are in flight.
func (c *Client) WithMaxRequestConcurrent(maxConn int) *Client {
	c.MaxConcurrentRequests = maxConn
	c.requestResultCh = make(chan requestResult, c.MaxConcurrentRequests)
	c.sendControlCh = make(chan struct{}, c.MaxConcurrentRequests)
	return c
}

// start sending requests.
func (c *Client) SendRequests() {
	if c.resolver != nil {
		ips, err := c.resolver.Lookup(time.Second, c.URL.Host)
		if err != nil {
			fmt.Printf("url %s host %s error %v\n", c.URL, c.URL.Host, err)
			return
		}
		c.availableIPs = ips
	} else {
		ips, err := net.LookupHost(c.URL.Host)
		if err != nil {
			return
		}
		c.availableIPs = ips
	}

	c.selector = NewRoundRobinSelector(c.availableIPs)

	// rate limit for concurrent requests
	go func() {
		c.sendControlCh <- struct{}{}
		ticker := time.NewTicker(c.RequestInterval)
		for range ticker.C {
			c.sendControlCh <- struct{}{}
		}
	}()

	fmt.Printf("url %s host %s ips %v\n", c.URL, c.URL.Host, c.availableIPs)
	for {
		select {
		case res := <-c.requestResultCh:
			if res.err != nil {
				// failed to get response.
				fmt.Printf("ip %s failed to get response: %v\n", res.ip, res.err)
				c.selector.SetFail(res.ip)
			} else if res.code >= 500 {
				// 5xx response.
				fmt.Printf("ip %s failed, code %v\n", res.ip, res.code)
				c.selector.SetFail(res.ip)
			} else {
				// a successful response.
				successRequests := atomic.AddInt32(&c.successRequests, 1)
				fmt.Printf("ip %s request %d success\n", res.ip, successRequests)
				if int(successRequests) >= c.Requests {
					return
				}
			}
		default:
			go func() {
				// wait for limit control
				<-c.sendControlCh
				c.sendOneRequest()
			}()
		}
	}
}

// send one request to server.
func (c *Client) sendOneRequest() {
	ip := c.selector.SelectIP()
	dialer := &net.Dialer{
		Timeout:   c.RequestTimeout,
		KeepAlive: 30 * time.Second,
	}
	httpClient := &http.Client{
		Timeout: c.RequestTimeout,
		Transport: &http.Transport{
			// set target IP to the selected IP by specifying dial function.
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, _ := net.SplitHostPort(addr)
				if port == "" {
					addr = host
				} else {
					addr = ip + ":" + port
				}
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}

	urlStr := c.URL.String()
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Host = c.URL.Host
	start := time.Now()
	resp, err := httpClient.Do(req)
	if err != nil {
		reqResult := requestResult{
			err: err,
			ip:  ip,
		}
		// notice an error.
		c.requestResultCh <- reqResult
		return
	}
	dur := time.Now().Sub(start)

	// notice the response code.
	reqResult := requestResult{
		code:        resp.StatusCode,
		ip:          ip,
		responseDur: dur,
	}
	c.requestResultCh <- reqResult

	if c.HandleResponse != nil {
		// handle response if wanted to use the response to do something.
		c.HandleResponse(resp)
	}
}
