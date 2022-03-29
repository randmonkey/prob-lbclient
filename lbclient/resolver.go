package lbclient

import (
	"context"
	"net"
	"time"
)

type Resolver struct {
	dnsServers []string
}

type combinedResolveRes struct {
	names []string
	err   error
}

func NewResolver(dnsServers []string) *Resolver {
	r := &Resolver{
		dnsServers: []string{},
	}
	for _, server := range dnsServers {
		host, port, _ := net.SplitHostPort(server)
		if port == "" {
			r.dnsServers = append(r.dnsServers, host+":"+"53")
		} else {
			r.dnsServers = append(r.dnsServers, host+":"+port)
		}
	}
	return r
}

func (r *Resolver) Lookup(timeout time.Duration, host string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if len(r.dnsServers) == 0 {
		return net.DefaultResolver.LookupHost(ctx, host)
	}

	resChan := make(chan *combinedResolveRes, 2)
	for _, server := range r.dnsServers {
		dnsResolver := &net.Resolver{
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Millisecond * time.Duration(10000),
				}
				return d.DialContext(ctx, network, server)
			},
		}

		go func() {
			names, err := dnsResolver.LookupHost(ctx, host)
			combinedResult := &combinedResolveRes{
				names: names,
				err:   err,
			}
			resChan <- combinedResult
		}()
	}
	for {
		select {
		case res := <-resChan:
			if res.err == nil {
				return res.names, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
