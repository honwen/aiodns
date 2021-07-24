package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/utils"
	"github.com/Workiva/go-datastructures/set"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v3"
)

var (
	options = Options{
		AllServers:       true,
		EnableEDNSSubnet: true,
		TLSMinVersion:    1.2,
	}

	defaultUpstream = new(cli.StringSlice)
	specUpstream    = new(cli.StringSlice)
	fallUpstream    = new(cli.StringSlice)
	bootUpstream    = new(cli.StringSlice)
)

func cliErrorExit(c *cli.Context, err error) {
	fmt.Printf("%+v", err)
	cli.ShowAppHelp(c)
	os.Exit(-1)
}

func fetch(uri string, resolvers []string) (dat []byte, err error) {
	// Fetch List (Online or Local)
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		log.Printf("Fetching online list: [%s]", uri)
		dat, err = curl(uri, resolvers, 5)
	} else {
		if strings.HasPrefix(uri, "~") {
			homedir, _ := os.UserHomeDir()
			uri = homedir + uri[1:]
		}
		log.Printf("Fetching local list: [%s]", uri)
		dat, err = ioutil.ReadFile(uri)
	}

	// gunzip if needed
	if strings.HasSuffix(uri, ".gz") {
		if zReader, zErr := gzip.NewReader(bytes.NewReader(dat)); zErr == nil {
			dat, _ = ioutil.ReadAll(zReader)
		} else {
			err = zErr
		}
	}
	return
}

func scanDoamins(dat []byte, filter func(string) bool) (domains *set.Set) {
	domains = set.New()
	scanner := bufio.NewScanner(bytes.NewReader(dat))
	for scanner.Scan() {
		it := strings.TrimSpace(scanner.Text())
		for strings.HasPrefix(it, "#") {
			continue
		}
		for strings.HasPrefix(it, ".") {
			it = it[1:]
		}
		for strings.HasSuffix(it, ".") && len(it) > 0 {
			it = it[:len(it)-1]
		}
		if match, _ := regexp.MatchString(`^(server|ipset)=/[^\/]*/`, it); match {
			it = it[8:strings.LastIndex(it, `/`)]
		}
		if len(it) <= 0 || (filter != nil && filter(it)) {
			continue
		}
		if utils.IsValidHostname(it) != nil {
			fmt.Printf("Domain Skiped: %s\n", it)
			continue
		}
		domains.Add(it)
	}
	return
}

func main() {
	app := cli.NewApp()
	app.Name = "AIO DNS"
	app.Usage = "All In One Clean DNS Solution."
	app.Version = fmt.Sprintf("Git:[%s] (%s)", VersionString, runtime.Version())

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
			Usage: "Fallback resolvers to use when regular ones are unavailable, can be specified multiple times",
		},
		cli.StringSliceFlag{
			Name:  "bootstrap, b",
			Value: bootUpstream,
			Usage: "Bootstrap DNS for DoH and DoT, can be specified multiple times",
		},
		cli.StringSliceFlag{
			Name:  "special-list, L",
			Usage: "List of domains using special-upstream (can be specified multiple times)",
		},
		cli.StringSliceFlag{
			Name:  "bypass-list, B",
			Usage: "List of domains bypass special-upstream (can be specified multiple times)",
		},
		cli.StringFlag{
			Name:  "edns, e",
			Usage: "Send EDNS Client Address to default upstreams",
		},
		cli.IntFlag{
			Name:  "timeout, t",
			Value: 3,
			Usage: "Timeout of Each upstream, [1, 59] seconds",
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
		if !strings.HasPrefix(VersionString, "undefined") {
			fmt.Fprintf(os.Stderr, "%s %s\n", strings.ToUpper(c.App.Name), c.App.Version)
		}

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

		if timeout := c.Int("timeout"); 0 < timeout && timeout < 60 {
			defaultTimeout = time.Duration(timeout) * time.Second
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
			options.CacheOptimistic = true           // Prefetch
		}

		options.Upstreams = c.StringSlice("upstream")
		options.Fallbacks = c.StringSlice("fallback")
		options.BootstrapDNS = c.StringSlice("bootstrap")

		specLists := []string{} // list[domains mulit-lines]
		if len(c.StringSlice("special-list")) > 0 {
			for _, it := range c.StringSlice("special-list") {
				dat, err := fetch(it, options.BootstrapDNS)

				// skip if error
				if err != nil {
					log.Println(err)
					log.Printf("Failed; Skipped! [%s]", it)
					continue
				}

				// append special-list
				specLists = append(specLists, string(dat))
				log.Printf("%d lines special list fetched", len(strings.Split(string(dat), "\n")))
			}
		}

		// FailSafe or Default
		if len(specLists) <= 0 {
			log.Printf("Using build-in special list")
			specLists = append(specLists, specList)
			specLists = append(specLists, tldnList)

			// tldn := scanDoamins([]byte(tldnList), nil)
			// tide := scanDoamins([]byte(specList), func(s string) bool {
			// 	for _, it := range tldn.Flatten() {
			// 		if strings.HasSuffix(s, "."+it.(string)) {
			// 			return true
			// 		}
			// 	}
			// 	return false
			// })
			// for _, it := range tide.Flatten() {
			// 	fmt.Println(it)
			// }

			// tldn := scanDoamins([]byte(tldnList), nil)
			// tide := scanDoamins([]byte(bypassList), func(s string) bool {
			// 	for _, it := range tldn.Flatten() {
			// 		if strings.HasSuffix(s, "."+it.(string)) {
			// 			return false
			// 		}
			// 	}
			// 	return true
			// })
			// for _, it := range tide.Flatten() {
			// 	fmt.Println(it)
			// }
		}

		specDomains := scanDoamins([]byte(strings.Join(specLists, "\n")), nil)

		for _, u := range c.StringSlice("special-upstream") {
			for _, it := range specDomains.Flatten() {
				nUpstream := fmt.Sprintf("[/%s/]%s", it, u)
				options.Upstreams = append(options.Upstreams, nUpstream)
			}
		}

		bypassDomains := set.New()
		if len(c.StringSlice("bypass-list")) > 0 {
			for _, it := range c.StringSlice("bypass-list") {
				dat, err := fetch(it, options.BootstrapDNS)

				// skip if error
				if err != nil {
					log.Println(err)
					log.Printf("Failed; Skipped! [%s]", it)
					continue
				}

				// append bypass-list
				bypassDomains.Add(scanDoamins(dat, nil).Flatten()...)
				log.Printf("%d lines bypass list fetched", len(strings.Split(string(dat), "\n")))
			}
		} else if len(c.StringSlice("special-list")) < 1 {
			// only use build-in bypassList if special-list NOT configured
			log.Printf("Using build-in bypass list")
			bypassDomains = scanDoamins([]byte(bypassList), nil)
		}

		for _, it := range bypassDomains.Flatten() {
			nUpstream := fmt.Sprintf("[/%s/]%s", it, `#`)
			options.Upstreams = append(options.Upstreams, nUpstream)
		}

		for _, u := range initSpecUpstreams {
			for _, it := range initSpecDomains.Flatten() {
				options.Upstreams = append(options.Upstreams, fmt.Sprintf("[/%s/]%s", it, u))
			}
		}

		if options.Verbose {
			dump, _ := yaml.Marshal(&options)
			fmt.Println(string(dump))
		} else {
			log.Printf("Speclist Length: %d", specDomains.Len())
			log.Printf("Bypasslist Length: %d", bypassDomains.Len())
			log.Printf("Upstream Rule Count: %d", len(options.Upstreams))
		}

		run(&options)
		return nil
	}
	app.Run(os.Args)
}
