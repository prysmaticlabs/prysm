// Package beacon-chain defines all the utlities needed for a beacon chain node.
package main

import (
	"os"
	"runtime"

	"github.com/prysmaticlabs/prysm/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func startNode(ctx *cli.Context) error {
	verbosity := ctx.GlobalString(cmd.VerbosityFlag.Name)
	level, err := logrus.ParseLevel(verbosity)
	if err != nil {
		return err
	}
	logrus.SetLevel(level)

	beacon, err := node.NewBeaconNode(ctx)
	if err != nil {
		return err
	}
	beacon.Start()
	return nil
}

func main() {
	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)
	log := logrus.WithField("prefix", "main")
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

	app.Flags = []cli.Flag{
		utils.DemoConfigFlag,
		utils.SimulatorFlag,
		utils.VrcContractFlag,
		utils.PubKeyFlag,
		utils.Web3ProviderFlag,
		utils.RPCPort,
		utils.CertFlag,
		utils.KeyFlag,
		utils.GenesisJSON,
		utils.EnablePOWChain,
		cmd.DataDirFlag,
		cmd.VerbosityFlag,
		cmd.EnableTracingFlag,
		cmd.TracingEndpointFlag,
		cmd.TraceSampleFractionFlag,
		debug.PProfFlag,
		debug.PProfAddrFlag,
		debug.PProfPortFlag,
		debug.MemProfileRateFlag,
		debug.CPUProfileFlag,
		debug.TraceFlag,
	}

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return debug.Setup(ctx)
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
