// Package beacon-chain defines the entire runtime of an eth2 beacon node.
package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	runtimeDebug "runtime/debug"
	"strings"

	gethlog "github.com/ethereum/go-ethereum/log"
	golog "github.com/ipfs/go-log"
	joonix "github.com/joonix/log"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/logutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	gologging "github.com/whyrusleeping/go-logging"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	_ "go.uber.org/automaxprocs"
	"gopkg.in/urfave/cli.v2"
	"gopkg.in/urfave/cli.v2/altsrc"
	"gopkg.in/yaml.v2"
)

var log *logrus.Entry
var appFlags = []cli.Flag{
	flags.DepositContractFlag,
	flags.Web3ProviderFlag,
	flags.HTTPWeb3ProviderFlag,
	flags.RPCHost,
	flags.RPCPort,
	flags.CertFlag,
	flags.KeyFlag,
	flags.GRPCGatewayPort,
	flags.MinSyncPeers,
	flags.RPCMaxPageSize,
	flags.ContractDeploymentBlock,
	flags.SetGCPercent,
	flags.UnsafeSync,
	flags.DisableDiscv5,
	flags.BlockBatchLimit,
	flags.InteropMockEth1DataVotesFlag,
	flags.InteropGenesisStateFlag,
	flags.InteropNumValidatorsFlag,
	flags.InteropGenesisTimeFlag,
	flags.ArchiveEnableFlag,
	flags.ArchiveValidatorSetChangesFlag,
	flags.ArchiveBlocksFlag,
	flags.ArchiveAttestationsFlag,
	flags.SlotsPerArchivedPoint,
	flags.EnableDebugRPCEndpoints,
	cmd.BootstrapNode,
	cmd.NoDiscovery,
	cmd.StaticPeers,
	cmd.RelayNode,
	cmd.P2PUDPPort,
	cmd.P2PTCPPort,
	cmd.P2PIP,
	cmd.P2PHost,
	cmd.P2PHostDNS,
	cmd.P2PMaxPeers,
	cmd.P2PPrivKey,
	cmd.P2PMetadata,
	cmd.P2PWhitelist,
	cmd.P2PEncoding,
	cmd.P2PPubsub,
	cmd.DataDirFlag,
	cmd.VerbosityFlag,
	cmd.EnableTracingFlag,
	cmd.TracingProcessNameFlag,
	cmd.TracingEndpointFlag,
	cmd.TraceSampleFractionFlag,
	flags.MonitoringPortFlag,
	cmd.DisableMonitoringFlag,
	cmd.ClearDB,
	cmd.ForceClearDB,
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
	cmd.ConfigFileFlag,
	cmd.ChainConfigFileFlag,
}

func init() {
	appFlags = cmd.WrapFlags(append(appFlags, featureconfig.BeaconChainFlags...))
	log = logrus.WithField("prefix", "main")
}

func main() {
	app := cli.App{}
	app.Name = "beacon-chain"
	app.Usage = "this is a beacon chain implementation for Ethereum 2.0"
	app.Action = startNode
	app.Version = version.GetVersion()

	app.Flags = appFlags

	app.Before = func(ctx *cli.Context) error {
		// Load any flags from file, if specified.
		if ctx.IsSet(cmd.ConfigFileFlag.Name) {
			if err := altsrc.InitInputSourceWithContext(appFlags, altsrc.NewYamlSourceFromFlagFunc(cmd.ConfigFileFlag.Name))(ctx); err != nil {
				return err
			}
		}

		format := ctx.String(cmd.LogFormat.Name)
		switch format {
		case "text":
			formatter := new(prefixed.TextFormatter)
			formatter.TimestampFormat = "2006-01-02 15:04:05"
			formatter.FullTimestamp = true
			// If persistent log files are written - we disable the log messages coloring because
			// the colors are ANSI codes and seen as gibberish in the log files.
			formatter.DisableColors = ctx.String(cmd.LogFileName.Name) != ""
			logrus.SetFormatter(formatter)
			break
		case "fluentd":
			f := joonix.NewFormatter()
			if err := joonix.DisableTimestampFormat(f); err != nil {
				panic(err)
			}
			logrus.SetFormatter(f)
			break
		case "json":
			logrus.SetFormatter(&logrus.JSONFormatter{})
			break
		default:
			return fmt.Errorf("unknown log format %s", format)
		}

		logFileName := ctx.String(cmd.LogFileName.Name)
		if logFileName != "" {
			if err := logutil.ConfigurePersistentLogging(logFileName); err != nil {
				log.WithError(err).Error("Failed to configuring logging to disk.")
			}
		}
		if ctx.IsSet(cmd.ChainConfigFileFlag.Name) {
			chainConfigFileName := ctx.String(cmd.ChainConfigFileFlag.Name)
			loadChainConfigFile(chainConfigFileName)
		}
		if ctx.IsSet(flags.SetGCPercent.Name) {
			runtimeDebug.SetGCPercent(ctx.Int(flags.SetGCPercent.Name))
		}
		runtime.GOMAXPROCS(runtime.NumCPU())
		return debug.Setup(ctx)
	}

	defer func() {
		if x := recover(); x != nil {
			log.Errorf("Runtime panic: %v\n%v", x, string(runtimeDebug.Stack()))
			panic(x)
		}
	}()

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func loadChainConfigFile(chainConfigFileName string) {
	yamlFile, err := ioutil.ReadFile(chainConfigFileName)
	if err != nil {
		log.WithError(err).Error("Failed to read chain config file.")
	}
	lines := strings.Split(string(yamlFile), "\n")

	for i, line := range lines {
		if strings.Contains(line, "0x") {

			parts := strings.Split(line, "0x")
			b, err := hex.DecodeString(parts[1])
			if err != nil {
				log.WithError(err).Error("Failed to decode hex string.")
			}
			switch l := len(b); {
			case l > 0 && l <= 4:
				var arr [4]byte
				copy(arr[:], b)
				fixedByte, err := yaml.Marshal(arr)
				if err != nil {
					log.WithError(err).Error("Failed to marshal config file.")
				}
				parts[1] = string(fixedByte)
			case l > 4 && l <= 8:
				var arr [8]byte
				copy(arr[:], b)
				fixedByte, err := yaml.Marshal(arr)
				if err != nil {
					log.WithError(err).Error("Failed to marshal config file.")
				}
				parts[1] = string(fixedByte)
			case l > 8 && l <= 32:
				var arr [32]byte
				copy(arr[:], b)
				fixedByte, err := yaml.Marshal(arr)
				if err != nil {
					log.WithError(err).Error("Failed to marshal config file.")
				}
				parts[1] = string(fixedByte)
			case l > 32 && l <= 48:
				var arr [48]byte
				copy(arr[:], b)
				fixedByte, err := yaml.Marshal(arr)
				if err != nil {
					log.WithError(err).Error("Failed to marshal config file.")
				}
				parts[1] = string(fixedByte)
			case l > 48 && l <= 64:
				var arr [48]byte
				copy(arr[:], b)
				fixedByte, err := yaml.Marshal(arr)
				if err != nil {
					log.WithError(err).Error("Failed to marshal config file.")
				}
				parts[1] = string(fixedByte)
			case l > 64 && l <= 96:
				var arr [48]byte
				copy(arr[:], b)
				fixedByte, err := yaml.Marshal(arr)
				if err != nil {
					log.WithError(err).Error("Failed to marshal config file.")
				}
				parts[1] = string(fixedByte)
			}

			lines[i] = strings.Join(parts, "\n")
		}
	}
	yamlFile = []byte(strings.Join(lines, "\n"))
	conf := params.BeaconConfig()
	if err := yaml.Unmarshal(yamlFile, conf); err != nil {
		log.WithError(err).Error("Failed to parse chain config yaml file.")
	}
	params.OverrideBeaconConfig(conf)
}

func startNode(ctx *cli.Context) error {
	verbosity := ctx.String(cmd.VerbosityFlag.Name)
	level, err := logrus.ParseLevel(verbosity)
	if err != nil {
		return err
	}
	logrus.SetLevel(level)
	if level == logrus.TraceLevel {
		// libp2p specific logging.
		golog.SetAllLoggers(gologging.DEBUG)
		// Geth specific logging.
		glogger := gethlog.NewGlogHandler(gethlog.StreamHandler(os.Stderr, gethlog.TerminalFormat(true)))
		glogger.Verbosity(gethlog.LvlTrace)
		gethlog.Root().SetHandler(glogger)
	}

	beacon, err := node.NewBeaconNode(ctx)
	if err != nil {
		return err
	}
	beacon.Start()
	return nil
}
