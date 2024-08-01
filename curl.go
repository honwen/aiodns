package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
)

var tcpTimeout = 30 * time.Second

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
	bootUpstreams := []upstream.Upstream{}
	for _, it := range resolvers {
		if b, err := upstream.AddressToUpstream(it, &upstream.Options{Timeout: tcpTimeout}); err == nil {
			bootUpstreams = append(bootUpstreams, b)
		}
	}

	if len(bootUpstreams) > 0 {
		bootUpstreamResolver, _ := proxy.New(&proxy.Config{
			UpstreamMode: proxy.UpstreamModeParallel,
			UpstreamConfig: &proxy.UpstreamConfig{
				Upstreams: bootUpstreams,
			},
		})

		defer func() {
			bootUpstreams = nil
			bootUpstreamResolver = nil
		}()

		client.Transport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, _ := net.SplitHostPort(addr)
			if addrs, err := bootUpstreamResolver.LookupNetIP(ctx, "ip", host); err == nil {
				for _, v := range addrs {
					if v.IsValid() {
						addr = net.JoinHostPort(v.String(), port)
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
	if resp, httpErr := client.Do(request); httpErr != nil {
		err = httpErr
		if retry <= 0 {
			return
		} else {
			return curl(url, resolvers, retry-1)
		}
	} else {
		data, err = io.ReadAll(resp.Body)
		resp.Body.Close()
	}
	return
}
