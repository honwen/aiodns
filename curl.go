package main

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
)

var (
	tcpTimeout = 30 * time.Second
)

func curl(url string, resolvers []string, retry int) (data []byte, err error) {
	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: tcpTimeout,
			IdleConnTimeout:       tcpTimeout,
			DisableKeepAlives:     true,
		},
		Timeout: tcpTimeout,
	}
	dialer := &net.Dialer{
		Timeout:   tcpTimeout,
		DualStack: true,
	}
	bootNSs := []*upstream.Resolver{}
	for _, it := range resolvers {
		if b, err := upstream.AddressToUpstream(it, &upstream.Options{Timeout: tcpTimeout}); err == nil {
			if r, err := upstream.NewResolver(b.Address(), &upstream.Options{Timeout: tcpTimeout}); err == nil {
				bootNSs = append(bootNSs, r)
			}
		}
	}
	if len(bootNSs) > 0 {
		client.Transport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, _ := net.SplitHostPort(addr)
			if addrs, err := upstream.LookupParallel(ctx, bootNSs, host); err == nil {
				for _, v := range addrs {
					if v.IP.To4() != nil || v.IP.To16() == nil {
						addr = net.JoinHostPort(v.IP.String(), port)
						break
					}
				}
			} else {
				return nil, err
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}

	request, _ := http.NewRequest("GET", url, nil)
	if resp, err := client.Do(request); err != nil {
		if retry <= 0 {
			return nil, err
		} else {
			return curl(url, resolvers, retry-1)
		}
	} else {
		data, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
	return
}
