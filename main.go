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

	"github.com/honwen/aiodns/autodns"
	"github.com/honwen/dnspod-http-dns/dnspod"
	"github.com/honwen/https-dns/gdns"
	"github.com/golang/glog"
	"github.com/miekg/dns"
	"github.com/urfave/cli"
)

var (
	version = "MISSING build version [git hash]"

	outHandler *gdns.Handler
	inHandler  *dnspod.DNSPOD

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
	rand.Seed(time.Now().UTC().UnixNano())
}

func init() {
	rand.Seed(time.Now().UnixNano())
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
		cli.BoolFlag{
			Name:  "insecure, I",
			Usage: "Disable SSL/TLS Certificate check (for some OS without ca-certificates)",
		},
		cli.StringFlag{
			Name:  "edns, e",
			Usage: "Extension mechanisms for DNS (EDNS) is parameters of the Domain Name System (DNS) protocol.",
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

		gdnsOPT := gdns.GDNSOptions{
			EndpointIPs: []net.IP{net.ParseIP("210.17.9.228")},
			EDNS:        c.String("edns"),
			Secure:      !c.Bool("insecure"),
		}
		gdnsEndPT := `https://dns.twnic.tw/dns-query`

		outProvider, err := gdns.NewGDNSProvider(gdnsEndPT, &gdnsOPT)
		if err != nil {
			glog.Exitln(err)
		}

		outHandler = gdns.NewHandler(outProvider, new(gdns.HandlerOptions))
		inHandler = dnspod.NewDNSPOD("")

		if !strings.HasPrefix(version, "MISSING") {
			fmt.Fprintf(os.Stderr, "%s %s\n", strings.ToUpper(c.App.Name), c.App.Version)
		}
		return nil
	}
	app.Flags = append(app.Flags, glogGangstaFlags...)
	app.Run(os.Args)
	defer glog.Flush()

	autoDNS := autodns.NewAutoDNS(inHandler.DNSHandleFunc, outHandler.Handle, "")
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
