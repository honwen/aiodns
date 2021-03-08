package main

//https://github.com/urfave/cli/issues/269

import (
	"flag"
	"fmt"

	"github.com/urfave/cli"
)

// Log level for glog
const (
	LFATAL = iota
	LERROR
	LWARNING
	LINFO
	LDEBUG
)

func glogFlagShim(fakeVals map[string]string) {
	flag.VisitAll(func(fl *flag.Flag) {
		if val, ok := fakeVals[fl.Name]; ok {
			fl.Value.Set(val)
		}
	})
}

func glogGangstaShim(c *cli.Context) {
	flag.CommandLine.Parse([]string{})
	glogFlagShim(map[string]string{
		"v":           fmt.Sprint(c.Int("V")),
		"logtostderr": fmt.Sprint(c.Bool("logtostderr")),
	})
}

var glogGangstaFlags = []cli.Flag{
	cli.IntFlag{
		Name: "V", Value: 2, Usage: "log level for V logs",
	},
	cli.BoolFlag{
		Name: "logtostderr", Usage: "log to standard error instead of files",
	},
}
