package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/AdguardTeam/golibs/log"
	"github.com/asaskevich/govalidator"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v3"
)

var (
	version = "MISSING build version [git hash]"

	options = Options{
		AllServers:       true,
		EnableEDNSSubnet: true,
	}

	defaultUpstream = new(cli.StringSlice)
	specUpstream    = new(cli.StringSlice)
	fallUpstream    = new(cli.StringSlice)
	bootUpstream    = new(cli.StringSlice)
)

func init() {
	defaultUpstream.Set("tls://dns.pub")
	defaultUpstream.Set("tls://223.6.6.6")
	defaultUpstream.Set("https://doh.pub/dns-query")
	defaultUpstream.Set("https://dns.alidns.com/dns-query")

	specUpstream.Set("tls://dns.google")
	specUpstream.Set("tls://162.159.36.1")
	// specUpstream.Set("tls://dns.adguard.com")
	// specUpstream.Set("quic://dns.adguard.com")
	specUpstream.Set("https://dns.google/dns-query")
	specUpstream.Set("https://doh.dns.sb/dns-query")
	specUpstream.Set("https://cloudflare-dns.com/dns-query")

	fallUpstream.Set("tls://dns.rubyfish.cn")
	fallUpstream.Set("https://i.233py.com/dns-query")
	fallUpstream.Set("https://dns.rubyfish.cn/dns-query")
	bootUpstream.Set("tls://223.5.5.5")
	bootUpstream.Set("tls://1.0.0.1")
	bootUpstream.Set("114.114.115.115")
}

func cliErrorExit(c *cli.Context, err error) {
	fmt.Printf("%+v", err)
	cli.ShowAppHelp(c)
	os.Exit(-1)
}

func main() {
	app := cli.NewApp()
	app.Name = "AIO DNS"
	app.Usage = "All In One Clean DNS Solution."
	app.Version = fmt.Sprintf("Git:[%s] (%s)", strings.ToUpper(version), runtime.Version())
	// app.HideVersion = true
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "listen, l",
			Value: ":5300",
			Usage: "Listening address",
		},
		cli.StringSliceFlag{
			Name:  "upstream, u",
			Value: defaultUpstream,
			Usage: "An upstream to be default used (can be specified multiple times)",
		},
		cli.StringSliceFlag{
			Name:  "special-upstream, U",
			Value: specUpstream,
			Usage: "An upstream to be special used (can be specified multiple times)",
		},
		cli.StringSliceFlag{
			Name:  "fallback, f",
			Value: fallUpstream,
			Usage: "Bootstrap DNS for DoH and DoT, can be specified multiple times",
		},
		cli.StringSliceFlag{
			Name:  "bootstrap, b",
			Value: bootUpstream,
			Usage: "Bootstrap DNS for DoH and DoT, can be specified multiple times",
		},
		cli.StringSliceFlag{
			Name:  "special-list, L",
			Usage: "List of domains  using special-upstream (can be specified multiple times)",
		},
		cli.StringFlag{
			Name:  "edns, e",
			Usage: "Send EDNS Client Address to default upstreams",
		},
		cli.BoolFlag{
			Name:  "cache, C",
			Usage: "If specified, DNS cache is enabled",
		},
		cli.BoolFlag{
			Name:  "insecure, I",
			Usage: "If specified, disable SSL/TLS Certificate check (for some OS without ca-certificates)",
		},
		cli.BoolFlag{
			Name:  "ipv6-disabled, R",
			Usage: "If specified, all AAAA requests will be replied with NoError RCode and empty answer",
		},
		cli.BoolFlag{
			Name:  "refuse-any, A",
			Usage: "If specified, refuse ANY requests",
		},
		cli.BoolFlag{
			Name:  "fastest-addr, F",
			Usage: "If specified, Respond to A or AAAA requests only with the fastest IP address",
		},
		cli.BoolFlag{
			Name:  "verbose, V",
			Usage: "If specified, Verbose output",
		},
	}

	app.Action = func(c *cli.Context) error {
		if host, port, err := net.SplitHostPort(c.String("listen")); err != nil {
			cliErrorExit(c, err)
		} else {
			if hostIP := net.ParseIP(host); hostIP != nil {
				options.ListenAddrs = append(options.ListenAddrs, host)
			} else {
				options.ListenAddrs = append(options.ListenAddrs, "0.0.0.0")
			}
			if portInt, err := strconv.Atoi(port); err == nil {
				options.ListenPorts = append(options.ListenPorts, portInt)
			} else {
				cliErrorExit(c, err)
			}
		}

		options.EDNSAddr = c.String("edns")
		options.Cache = c.BoolT("cache")
		options.Verbose = c.BoolT("verbose")
		options.Insecure = c.BoolT("insecure")
		options.RefuseAny = c.BoolT("refuse-any")
		options.IPv6Disabled = c.BoolT("ipv6-disabled")
		options.FastestAddress = c.BoolT("fastest-addr")
		if options.FastestAddress {
			options.Cache = true
			options.CacheMinTTL = 600
		}
		if options.Cache {
			options.CacheSizeBytes = 4 * 1024 * 1024 // 4M
		}

		options.Upstreams = append(c.StringSlice("upstream"), initSpecUpstreams...)
		options.Fallbacks = c.StringSlice("fallback")
		options.BootstrapDNS = c.StringSlice("bootstrap")

		specUpstreams := map[string]bool{}

		specLists := []string{} // list[domains mulit-lines]
		if len(c.StringSlice("special-list")) > 0 {
			for _, it := range c.StringSlice("special-list") {
				if dat, err := ioutil.ReadFile(it); err == nil {
					specLists = append(specLists, string(dat))
				}
			}
		} else {
			log.Printf("Using build-in special list")
			specLists = append(specLists, specList)
			specLists = append(specLists, tldnList)
		}
		for _, v := range specLists {
			specScanner := bufio.NewScanner(bytes.NewReader([]byte(v)))
			for specScanner.Scan() {
				it := strings.TrimSpace(specScanner.Text())
				for strings.HasPrefix(it, ".") {
					it = it[1:]
				}
				if len(it) <= 0 {
					continue
				}
				specUpstreams[it] = true
			}
		}

		for _, u := range c.StringSlice("special-upstream") {
			for it := range specUpstreams {
				nUpstream := fmt.Sprintf("[/%s/]%s", it, u)
				if !govalidator.IsDNSName(it) {
					log.Printf("Speclist Rule Skiped: %s", nUpstream)
					continue
				}
				options.Upstreams = append(options.Upstreams, nUpstream)
			}
		}

		if !strings.HasPrefix(version, "MISSING") {
			fmt.Fprintf(os.Stderr, "%s %s\n", strings.ToUpper(c.App.Name), c.App.Version)
		}

		if options.Verbose {
			dump, _ := yaml.Marshal(&options)
			fmt.Println(string(dump))
		} else {
			log.Printf("Speclist Length: %d", len(specUpstreams))
			log.Printf("Upstream Rule Count: %d", len(options.Upstreams))
		}

		run(options)
		return nil
	}
	app.Run(os.Args)
}
