// Package main defines slasher server implementation for Ethereum. A slasher
// listens for all broadcasted messages using a running beacon node in order
// to detect malicious attestations and block proposals.
package main

import (
	"fmt"
	"os"
	"runtime"

	joonix "github.com/joonix/log"
	"github.com/prysmaticlabs/prysm/cmd/slasher/flags"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/journald"
	"github.com/prysmaticlabs/prysm/shared/logutil"
	"github.com/prysmaticlabs/prysm/shared/tos"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/node"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func startSlasher(cliCtx *cli.Context) error {
	// verify if ToS accepted
	if err := tos.VerifyTosAcceptedOrPrompt(cliCtx); err != nil {
		return err
	}

	verbosity := cliCtx.String(cmd.VerbosityFlag.Name)
	level, err := logrus.ParseLevel(verbosity)
	if err != nil {
		return err
	}
	logrus.SetLevel(level)
	slasher, err := node.New(cliCtx)
	if err != nil {
		return err
	}
	slasher.Start()
	return nil
}

var appFlags = []cli.Flag{
	cmd.MinimalConfigFlag,
	cmd.E2EConfigFlag,
	cmd.RPCMaxPageSizeFlag,
	cmd.VerbosityFlag,
	cmd.DataDirFlag,
	cmd.EnableTracingFlag,
	cmd.TracingProcessNameFlag,
	cmd.TracingEndpointFlag,
	cmd.TraceSampleFractionFlag,
	cmd.MonitoringHostFlag,
	flags.MonitoringPortFlag,
	cmd.DisableMonitoringFlag,
	cmd.EnableBackupWebhookFlag,
	cmd.BackupWebhookOutputDir,
	cmd.LogFileName,
	cmd.LogFormat,
	cmd.ClearDB,
	cmd.ForceClearDB,
	cmd.ConfigFileFlag,
	debug.PProfFlag,
	debug.PProfAddrFlag,
	debug.PProfPortFlag,
	debug.MemProfileRateFlag,
	debug.CPUProfileFlag,
	debug.TraceFlag,
	flags.RPCPort,
	flags.RPCHost,
	flags.CertFlag,
	flags.KeyFlag,
	flags.BeaconCertFlag,
	flags.BeaconRPCProviderFlag,
	flags.EnableHistoricalDetectionFlag,
	flags.SpanCacheSize,
	cmd.AcceptTosFlag,
	flags.HighestAttCacheSize,
}

func init() {
	appFlags = cmd.WrapFlags(append(appFlags, featureconfig.SlasherFlags...))
}

func main() {
	app := cli.App{}
	app.Name = "hash slinging slasher"
	app.Usage = `launches an Ethereum Serenity slasher server that interacts with a beacon chain.`
	app.Version = version.Version()
	app.Commands = []*cli.Command{
		db.DatabaseCommands,
	}
	app.Flags = appFlags
	app.Action = startSlasher
	app.Before = func(ctx *cli.Context) error {
		// Load flags from config file, if specified.
		if err := cmd.LoadFlagsFromConfig(ctx, app.Flags); err != nil {
			return err
		}

		format := ctx.String(cmd.LogFormat.Name)
		switch format {
		case "text":
			formatter := new(prefixed.TextFormatter)
			formatter.TimestampFormat = "2006-01-02 15:04:05"
			formatter.FullTimestamp = true
			// If persistent log files are written - we disable the log messages coloring because
			// the colors are ANSI codes and seen as Gibberish in the log files.
			formatter.DisableColors = ctx.String(cmd.LogFileName.Name) != ""
			logrus.SetFormatter(formatter)
		case "fluentd":
			logrus.SetFormatter(joonix.NewFormatter())
		case "json":
			logrus.SetFormatter(&logrus.JSONFormatter{})
		case "journald":
			if err := journald.Enable(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown log format %s", format)
		}

		logFileName := ctx.String(cmd.LogFileName.Name)
		if logFileName != "" {
			if err := logutil.ConfigurePersistentLogging(logFileName); err != nil {
				log.WithError(err).Error("Failed to configuring logging to disk.")
			}
		}

		runtime.GOMAXPROCS(runtime.NumCPU())
		if err := debug.Setup(ctx); err != nil {
			return err
		}
		return cmd.ValidateNoArgs(ctx)
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
