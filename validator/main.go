package main

import (
	"os"
	"runtime"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/validator/node"
	"github.com/prysmaticlabs/prysm/validator/types"
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

	shardingNode, err := node.NewValidatorClient(ctx)
	if err != nil {
		return err
	}

	shardingNode.Start()
	return nil
}

func main() {
	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)
	log := logrus.WithField("prefix", "main")

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

	app := cli.NewApp()
	app.Name = "validator"
	app.Usage = `launches an Ethereum Serenity validator client that interacts with a beacon chain, starts proposer services, shardp2p connections, and more
`
	app.Version = version.GetVersion()
	app.Action = startNode
	app.Flags = []cli.Flag{
		types.BeaconRPCProviderFlag,
		types.PubKeyFlag,
		cmd.VerbosityFlag,
		cmd.DataDirFlag,
		cmd.EnableTracingFlag,
		cmd.TracingEndpointFlag,
		cmd.TraceSampleFractionFlag,
		cmd.KeystorePasswordFlag,
		cmd.KeystoreDirectoryFlag,
		cmd.BootstrapNode,
		cmd.MonitoringPortFlag,
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

	app.After = func(ctx *cli.Context) error {
		debug.Exit(ctx)
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
