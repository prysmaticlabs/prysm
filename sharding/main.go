package main

import (
	"os"
	"runtime"

	"github.com/prysmaticlabs/geth-sharding/sharding/node"
	"github.com/prysmaticlabs/geth-sharding/sharding/utils"
	"github.com/prysmaticlabs/geth-sharding/shared/cmd"
	"github.com/prysmaticlabs/geth-sharding/shared/debug"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func startNode(ctx *cli.Context) error {
	shardingNode, err := node.New(ctx)
	if err != nil {
		return err
	}

	shardingNode.Start()
	return nil
}

func main() {
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
	app.Name = "sharding"
	app.Usage = `launches a sharding client that interacts with a beacon chain, starts proposer services, shardp2p connections, and more
`
	app.Action = startNode
	app.Flags = []cli.Flag{utils.ActorFlag, cmd.DataDirFlag, cmd.PasswordFileFlag, cmd.NetworkIdFlag, cmd.IPCPathFlag, utils.DepositFlag, utils.ShardIDFlag, debug.PProfFlag, debug.PProfAddrFlag, debug.PProfPortFlag, debug.MemProfileRateFlag, debug.CPUProfileFlag, debug.TraceFlag}

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return debug.Setup(ctx)
	}

	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
