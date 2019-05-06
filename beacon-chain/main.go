// Package beacon-chain defines all the utlities needed for a beacon chain node.
package main

import (
	"os"
	"runtime"

	"github.com/prysmaticlabs/prysm/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/version"
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
	app.Name = "beacon-chain"
	app.Usage = "this is a beacon chain implementation for Ethereum 2.0"
	app.Action = startNode
	app.Version = version.GetVersion()

	app.Flags = []cli.Flag{
		utils.NoCustomConfigFlag,
		utils.DepositContractFlag,
		utils.Web3ProviderFlag,
		utils.HTTPWeb3ProviderFlag,
		utils.RPCPort,
		utils.CertFlag,
		utils.KeyFlag,
		utils.EnableDBCleanup,
		cmd.BootstrapNode,
		cmd.RelayNode,
		cmd.P2PPort,
		cmd.P2PHost,
		cmd.DataDirFlag,
		cmd.VerbosityFlag,
		cmd.EnableTracingFlag,
		cmd.TracingEndpointFlag,
		cmd.TraceSampleFractionFlag,
		cmd.MonitoringPortFlag,
		cmd.DisableMonitoringFlag,
		cmd.ClearDB,
		debug.PProfFlag,
		debug.PProfAddrFlag,
		debug.PProfPortFlag,
		debug.MemProfileRateFlag,
		debug.CPUProfileFlag,
		debug.TraceFlag,
	}

	app.Flags = append(app.Flags, featureconfig.BeaconChainFlags...)

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return debug.Setup(ctx)
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
