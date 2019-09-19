// Package beacon-chain defines all the utilities needed for a beacon chain node.
package main

import (
	"fmt"
	"os"
	"runtime"

	golog "github.com/ipfs/go-log"
	joonix "github.com/joonix/log"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/logutil"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	gologging "github.com/whyrusleeping/go-logging"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	_ "go.uber.org/automaxprocs"
)

var appFlags = []cli.Flag{
	flags.NoCustomConfigFlag,
	flags.DepositContractFlag,
	flags.Web3ProviderFlag,
	flags.HTTPWeb3ProviderFlag,
	flags.RPCPort,
	flags.CertFlag,
	flags.KeyFlag,
	flags.GRPCGatewayPort,
	flags.InteropMockEth1DataVotesFlag,
	flags.InteropGenesisStateFlag,
	flags.InteropNumValidatorsFlag,
	flags.InteropGenesisTimeFlag,
	flags.ArchiveEnableFlag,
	flags.ArchiveValidatorSetChangesFlag,
	flags.ArchiveBlocksFlag,
	flags.ArchiveAttestationsFlag,
	cmd.BootstrapNode,
	cmd.NoDiscovery,
	cmd.StaticPeers,
	cmd.RelayNode,
	cmd.P2PUDPPort,
	cmd.P2PTCPPort,
	cmd.P2PHost,
	cmd.P2PMaxPeers,
	cmd.P2PPrivKey,
	cmd.P2PWhitelist,
	cmd.P2PEncoding,
	cmd.DataDirFlag,
	cmd.VerbosityFlag,
	cmd.EnableTracingFlag,
	cmd.TracingProcessNameFlag,
	cmd.TracingEndpointFlag,
	cmd.TraceSampleFractionFlag,
	cmd.MonitoringPortFlag,
	cmd.DisableMonitoringFlag,
	cmd.ClearDB,
	cmd.LogFormat,
	cmd.MaxGoroutines,
	debug.PProfFlag,
	debug.PProfAddrFlag,
	debug.PProfPortFlag,
	debug.MemProfileRateFlag,
	debug.CPUProfileFlag,
	debug.TraceFlag,
	cmd.LogFileName,
	cmd.EnableUPnPFlag,
}

func init() {
	appFlags = append(appFlags, featureconfig.BeaconChainFlags...)
}

func main() {
	log := logrus.WithField("prefix", "main")
	app := cli.NewApp()
	app.Name = "beacon-chain"
	app.Usage = "this is a beacon chain implementation for Ethereum 2.0"
	app.Action = startNode
	app.Version = version.GetVersion()

	app.Flags = appFlags

	app.Before = func(ctx *cli.Context) error {
		format := ctx.GlobalString(cmd.LogFormat.Name)
		switch format {
		case "text":
			formatter := new(prefixed.TextFormatter)
			formatter.TimestampFormat = "2006-01-02 15:04:05"
			formatter.FullTimestamp = true
			// If persistent log files are written - we disable the log messages coloring because
			// the colors are ANSI codes and seen as gibberish in the log files.
			formatter.DisableColors = ctx.GlobalString(cmd.LogFileName.Name) != ""
			logrus.SetFormatter(formatter)
			break
		case "fluentd":
			logrus.SetFormatter(joonix.NewFormatter())
			break
		case "json":
			logrus.SetFormatter(&logrus.JSONFormatter{})
			break
		default:
			return fmt.Errorf("unknown log format %s", format)
		}

		logFileName := ctx.GlobalString(cmd.LogFileName.Name)
		if logFileName != "" {
			if err := logutil.ConfigurePersistentLogging(logFileName); err != nil {
				log.WithError(err).Error("Failed to configuring logging to disk.")
			}
		}

		runtime.GOMAXPROCS(runtime.NumCPU())
		return debug.Setup(ctx)
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func startNode(ctx *cli.Context) error {
	verbosity := ctx.GlobalString(cmd.VerbosityFlag.Name)
	level, err := logrus.ParseLevel(verbosity)
	if err != nil {
		return err
	}
	logrus.SetLevel(level)
	if level == logrus.DebugLevel {
		golog.SetAllLoggers(gologging.DEBUG)
	}

	beacon, err := node.NewBeaconNode(ctx)
	if err != nil {
		return err
	}
	beacon.Start()
	return nil
}
