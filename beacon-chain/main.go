// Package beacon-chain defines all the utilities needed for a beacon chain node.
package main

import (
	"fmt"
	"os"
	"runtime"

	joonix "github.com/joonix/log"
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

//---------------file logging hook code------------

// WriterHook is a hook that writes logs of specified LogLevels to specified Writer
type WriterHook struct {
	LogLevels []logrus.Level
}

// Fire will be called when some logging function is called with current hook
// It will format log entry to string and write it to appropriate writer
func (hook *WriterHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	//simply call the file logger Println func
	fileLogger.Println(line)
	return err
}

// Levels define on which log levels this hook would trigger
func (hook *WriterHook) Levels() []logrus.Level {
	return hook.LogLevels
}

var fileLogger = &logrus.Logger{
	Level: logrus.TraceLevel,
}

//------- end of file logging hook code---------------

func main() {
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
		cmd.NoDiscovery,
		cmd.StaticPeers,
		cmd.RelayNode,
		cmd.P2PPort,
		cmd.P2PHost,
		cmd.P2PMaxPeers,
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
		cmd.LogFileFormat,
	}

	app.Flags = append(app.Flags, featureconfig.BeaconChainFlags...)

	app.Before = func(ctx *cli.Context) error {
		format := ctx.GlobalString(cmd.LogFormat.Name)
		switch format {
		case "text":
			formatter := new(prefixed.TextFormatter)
			formatter.TimestampFormat = "2006-01-02 15:04:05"
			formatter.FullTimestamp = true
			formatter.DisableColors = ctx.GlobalString(cmd.LogFileFormat.Name) != ""
			logrus.SetFormatter(formatter)
			break
		case "fluentd":
			logrus.SetFormatter(&joonix.FluentdFormatter{})
			break
		case "json":
			logrus.SetFormatter(&logrus.JSONFormatter{})
			break
		default:
			return fmt.Errorf("unknown log format %s", format)
		}

		//if the user has specified a log file name (--log-file arg) -
		//we configure a persistent log file logger , a formatter and a hook.
		logFileName := ctx.GlobalString(cmd.LogFileName.Name)
		logrus.Info("Logs will be made persistent , logFileName=" + logFileName)
		if logFileName != "" {
			f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				logrus.Fatalf("Cannot open log file " + err.Error())
			}
			fileLogger.SetOutput(f)

			//configure format if specified, othereise use the stdout logger's format
			logFileFormatName := ctx.GlobalString(cmd.LogFileFormat.Name)
			switch logFileFormatName {
			case "text":
				formatter := new(prefixed.TextFormatter)
				formatter.TimestampFormat = "2006-01-02 15:04:05"
				formatter.FullTimestamp = true
				formatter.DisableColors = true
				fileLogger.SetFormatter(formatter)
				break
			case "fluentd":
				fileLogger.SetFormatter(&joonix.FluentdFormatter{})
				break
			case "json":
				fileLogger.SetFormatter(&logrus.JSONFormatter{})
				break
			default:
				logrus.Fatalf("must specifiy log file format when logging to persistent log file.")
			}

			logrus.Info("File logger initialized")
			//trigger writing to the log file on every stdout log write
			logrus.AddHook(&WriterHook{
				LogLevels: []logrus.Level{
					logrus.PanicLevel,
					logrus.FatalLevel,
					logrus.ErrorLevel,
					logrus.WarnLevel,
					logrus.InfoLevel,
					logrus.DebugLevel,
					logrus.TraceLevel,
				},
			})
		}

		runtime.GOMAXPROCS(runtime.NumCPU())
		return debug.Setup(ctx)
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
