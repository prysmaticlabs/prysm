package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/prysmaticlabs/geth-sharding/sharding/node"
	"github.com/prysmaticlabs/geth-sharding/sharding/utils"
	"github.com/urfave/cli"
)

func startNode(ctx *cli.Context) error {
	shardingNode, err := node.New(ctx)
	if err != nil {
		return err
	}
	// starts a connection to a beacon node and kicks off every registered service.
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
	app.Flags = []cli.Flag{utils.ActorFlag, utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag, utils.DepositFlag, utils.ShardIDFlag, utils.PProfFlag, utils.PProfAddrFlag, utils.PProfPortFlag, utils.MemProfileRateFlag, utils.CPUProfileFlag, utils.TraceFlag}

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		if err := utils.Setup(ctx); err != nil {
			return err
		}
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		utils.Exit()
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		if _, err := fmt.Fprintln(os.Stderr, err); err != nil {
			panic(err)
		}
		os.Exit(1)
	}
}
