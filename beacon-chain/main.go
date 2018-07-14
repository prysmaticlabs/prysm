package main

import (
	"os"
	"runtime"

	"github.com/prysmaticlabs/geth-sharding/beacon-chain/node"
	"github.com/prysmaticlabs/geth-sharding/shared/debug"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func startNode(ctx *cli.Context) error {
	beacon, err := node.New(ctx)
	if err != nil {
		return err
	}
	beacon.Start()
	return nil
}

func main() {
	app := cli.NewApp()
	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
`
	app.Name = "beacon-chain"
	app.Usage = "this is a beacon chain implementation for Ethereum 2.0"
	app.Action = startNode

	app.Flags = []cli.Flag{debug.PProfFlag, debug.PProfAddrFlag, debug.PProfPortFlag, debug.MemProfileRateFlag, debug.CPUProfileFlag, debug.TraceFlag}

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		if err := debug.Setup(ctx); err != nil {
			return err
		}
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
