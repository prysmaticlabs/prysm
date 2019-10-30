package main

import (
	"fmt"
	"os"
	"runtime"

	joonix "github.com/joonix/log"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/logutil"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/slasher/flags"
	"github.com/prysmaticlabs/prysm/slasher/service"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var log = logrus.WithField("prefix", "main")

func startSlasher(ctx *cli.Context) error {
	verbosity := ctx.GlobalString(cmd.VerbosityFlag.Name)
	level, err := logrus.ParseLevel(verbosity)
	if err != nil {
		return err
	}
	logrus.SetLevel(level)
	port := ctx.GlobalString(flags.RPCPort.Name)
	cert := ctx.GlobalString(flags.CertFlag.Name)
	key := ctx.GlobalString(flags.KeyFlag.Name)
	cfg := service.Config{
		Port:     port,
		CertFlag: cert,
		KeyFlag:  key,
	}
	slasher, err := service.NewRPCService(&cfg, ctx)
	if err != nil {
		return err
	}
	slasher.Start()
	return nil
}

var appFlags = []cli.Flag{
	cmd.VerbosityFlag,
	cmd.LogFormat,
	cmd.DataDirFlag,
	cmd.VerbosityFlag,
	cmd.DataDirFlag,
	cmd.EnableTracingFlag,
	cmd.TracingProcessNameFlag,
	cmd.TracingEndpointFlag,
	cmd.TraceSampleFractionFlag,
	cmd.BootstrapNode,
	cmd.MonitoringPortFlag,
	cmd.LogFileName,
	cmd.LogFormat,
	debug.PProfFlag,
	debug.PProfAddrFlag,
	debug.PProfPortFlag,
	debug.MemProfileRateFlag,
	debug.CPUProfileFlag,
	debug.TraceFlag,
	flags.CertFlag,
	flags.RPCPort,
	flags.KeyFlag,
}

func init() {

}

func main() {
	app := cli.NewApp()
	app.Name = "hash slinging slasher"
	app.Usage = `launches an Ethereum Serenity slasher server that interacts with a beacon chain.`
	app.Version = version.GetVersion()
	app.Action = startSlasher
	app.Flags = appFlags

	app.Before = func(ctx *cli.Context) error {
		format := ctx.GlobalString(cmd.LogFormat.Name)
		switch format {
		case "text":
			formatter := new(prefixed.TextFormatter)
			formatter.TimestampFormat = "2006-01-02 15:04:05"
			formatter.FullTimestamp = true
			// If persistent log files are written - we disable the log messages coloring because
			// the colors are ANSI codes and seen as Gibberish in the log files.
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

	app.After = func(ctx *cli.Context) error {
		debug.Exit(ctx)
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
