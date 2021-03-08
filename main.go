package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/golang/glog"
	"github.com/honwen/aiodns/autodns"
	"github.com/miekg/dns"
	"github.com/urfave/cli"
)

var (
	version = "MISSING build version [git hash]"

	defaultInUpstrem  = new(cli.StringSlice)
	defaultOutUpstrem = new(cli.StringSlice)
	defaultBootstraps = new(cli.StringSlice)

	outHandler *Handler
	inHandler  *Handler

	listenAddress   string
	listenProtocols []string
)

func serve(net, addr string) {
	glog.V(LINFO).Infof("starting %s service on %s", net, addr)

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	server := &dns.Server{Addr: addr, Net: net, TsigSecret: nil}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			glog.Errorf("Failed to setup the %s server: %s\n", net, err.Error())
			sig <- syscall.SIGTERM
		}
	}()

	// serve until exit
	<-sig

	glog.V(LINFO).Infof("shutting down %s on interrupt\n", net)
	if err := server.Shutdown(); err != nil {
		glog.V(LERROR).Infof("got unexpected error %s", err.Error())
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())

	defaultInUpstrem.Set("tls://dns.pub")
	defaultInUpstrem.Set("tls://223.6.6.6")
	defaultInUpstrem.Set("https://doh.pub/dns-query")
	defaultInUpstrem.Set("https://dns.alidns.com/dns-query")

	defaultOutUpstrem.Set("tls://dns.google")
	defaultOutUpstrem.Set("tls://162.159.36.1")
	// defaultOutUpstrem.Set("tls://dns.adguard.com")
	// defaultOutUpstrem.Set("quic://dns.adguard.com")
	defaultOutUpstrem.Set("https://dns.google/dns-query")
	defaultOutUpstrem.Set("https://doh.dns.sb/dns-query")
	defaultOutUpstrem.Set("https://cloudflare-dns.com/dns-query")

	defaultBootstraps.Set("tls://223.5.5.5")
	defaultBootstraps.Set("tls://1.0.0.1")
	defaultBootstraps.Set("114.114.115.115")
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
			Usage: "Serve address",
		},
		cli.StringSliceFlag{
			Name:  "ou, o",
			Value: defaultOutUpstrem,
			Usage: "Outside Upstreams",
		},
		cli.StringSliceFlag{
			Name:  "iu, i",
			Value: defaultInUpstrem,
			Usage: "Inside Upstreams",
		},
		cli.StringSliceFlag{
			Name:  "bootstrap, b",
			Value: defaultBootstraps,
			Usage: "Bootstrap Upstreams",
		},
		cli.BoolFlag{
			Name:  "insecure, I",
			Usage: "If specified, disable SSL/TLS Certificate check (for some OS without ca-certificates)",
		},
		cli.BoolFlag{
			Name:  "ipv6-disabled",
			Usage: "If specified, all AAAA requests will be replied with NoError RCode and empty answer",
		},
		cli.BoolFlag{
			Name:  "refuse-any",
			Usage: "If specified, refuse ANY requests",
		},
		cli.StringFlag{
			Name:  "out-edns, oe",
			Usage: "Send EDNS Client Address to Outside Upstreams",
		},
		cli.StringFlag{
			Name:  "in-edns, ie",
			Usage: "Send EDNS Client Address to Inside Upstreams",
		},

		cli.BoolFlag{
			Name:  "udp, U",
			Usage: "Listen on UDP",
		},
		cli.BoolFlag{
			Name:  "tcp, T",
			Usage: "Listen on TCP",
		},
	}
	app.Action = func(c *cli.Context) error {
		glogGangstaShim(c)
		listenAddress = c.String("listen")
		if c.Bool("tcp") {
			listenProtocols = append(listenProtocols, "tcp")
		}
		if c.Bool("udp") {
			listenProtocols = append(listenProtocols, "udp")
		}
		if 0 == len(listenProtocols) {
			cli.ShowAppHelp(c)
			os.Exit(0)
		}

		if !strings.HasPrefix(version, "MISSING") {
			fmt.Fprintf(os.Stderr, "%s %s\n", strings.ToUpper(c.App.Name), c.App.Version)
		}

		upstreamOptions := upstream.Options{
			Bootstrap:          c.StringSlice("bootstrap"),
			Timeout:            3333 * time.Millisecond,
			InsecureSkipVerify: c.Bool("insecure"),
		}

		handlerOptions := HandlerOptions{
			blockANY:  c.Bool("refuse-any"),
			blockAAAA: c.Bool("ipv6-disabled"),
		}

		var (
			ou   = c.StringSlice("ou")
			iu   = c.StringSlice("iu")
			outs []upstream.Upstream
			ins  []upstream.Upstream
		)

		for i := range ou {
			if out, err := upstream.AddressToUpstream(ou[i], upstreamOptions); err == nil {
				outs = append(outs, out)
				glog.V(LINFO).Infof("parse outside-upstream: %s", ou[i])
			} else {
				glog.V(LWARNING).Infof("fail to parse outside-upstream %s", ou[i])
				glog.V(LWARNING).Infoln(err)
			}
		}
		if len(outs) > 0 {
			hOptions := handlerOptions
			edns := c.String("out-edns")
			if ednsIP := net.ParseIP(edns); ednsIP != nil {
				glog.V(LDEBUG).Infof("parsed out-EDNS: %s", edns)
				hOptions.edns = &ednsIP
			} else {
				if len(edns) > 0 {
					glog.V(LWARNING).Infof("cannot parse out-EDNS: %s", edns)
				}
			}
			outHandler = NewHandler(outs, hOptions)
		} else {
			glog.V(LDEBUG).Infoln("No Valid outside-upstrea")
			os.Exit(0)
		}

		for i := range iu {
			if in, err := upstream.AddressToUpstream(iu[i], upstreamOptions); err == nil {
				ins = append(ins, in)
				glog.V(LINFO).Infof("parse inside-upstream: %s", iu[i])
			} else {
				glog.V(LWARNING).Infof("fail to parse inside-upstream %s", iu[i])
				glog.V(LWARNING).Infoln(err)
			}
		}
		if len(ins) > 0 {
			hOptions := handlerOptions
			edns := c.String("in-edns")
			if ednsIP := net.ParseIP(edns); ednsIP != nil {
				glog.V(LDEBUG).Infof("parsed in-EDNS: %s", edns)
				hOptions.edns = &ednsIP
			} else {
				if len(edns) > 0 {
					glog.V(LWARNING).Infof("cannot parse in-EDNS: %s", edns)
				}
			}
			inHandler = NewHandler(ins, hOptions)
		} else {
			glog.V(LDEBUG).Infoln("No Valid inside-upstrea")
			os.Exit(0)
		}

		return nil
	}
	app.Flags = append(app.Flags, glogGangstaFlags...)
	app.Run(os.Args)
	defer glog.Flush()

	autoDNS := autodns.NewAutoDNS(inHandler.HandleFunc, outHandler.HandleFunc, "")
	dns.HandleFunc(".", autoDNS.HandleFunc)

	// start the servers
	servers := make(chan bool)
	for _, protocol := range listenProtocols {
		go func(protocol string) {
			serve(protocol, listenAddress)
			servers <- true
		}(protocol)
	}

	// wait for servers to exit
	for range listenProtocols {
		<-servers
	}

	glog.V(LINFO).Infoln("servers exited, stopping")
}
