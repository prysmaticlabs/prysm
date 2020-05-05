// Package main defines a validator client, a critical actor in eth2 which manages
// a keystore of private keys, connects to a beacon node to receive assignments,
// and submits blocks/attestations as needed.
package main

import (
	"fmt"
	"os"
	"runtime"
	runtimeDebug "runtime/debug"

	joonix "github.com/joonix/log"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/logutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/node"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	_ "go.uber.org/automaxprocs"
	"gopkg.in/urfave/cli.v2"
	"gopkg.in/urfave/cli.v2/altsrc"
)

var log = logrus.WithField("prefix", "main")

func startNode(ctx *cli.Context) error {
	validatorClient, err := node.NewValidatorClient(ctx)
	if err != nil {
		return err
	}
	validatorClient.Start()
	return nil
}

var appFlags = []cli.Flag{
	flags.BeaconRPCProviderFlag,
	flags.CertFlag,
	flags.GraffitiFlag,
	flags.KeystorePathFlag,
	flags.PasswordFlag,
	flags.DisablePenaltyRewardLogFlag,
	flags.UnencryptedKeysFlag,
	flags.InteropStartIndex,
	flags.InteropNumValidators,
	flags.GrpcMaxCallRecvMsgSizeFlag,
	flags.GrpcRetriesFlag,
	flags.GrpcHeadersFlag,
	flags.KeyManager,
	flags.KeyManagerOpts,
	flags.AccountMetricsFlag,
	cmd.VerbosityFlag,
	cmd.DataDirFlag,
	cmd.ClearDB,
	cmd.ForceClearDB,
	cmd.EnableTracingFlag,
	cmd.TracingProcessNameFlag,
	cmd.TracingEndpointFlag,
	cmd.TraceSampleFractionFlag,
	flags.MonitoringPortFlag,
	cmd.LogFormat,
	debug.PProfFlag,
	debug.PProfAddrFlag,
	debug.PProfPortFlag,
	debug.MemProfileRateFlag,
	debug.CPUProfileFlag,
	debug.TraceFlag,
	cmd.LogFileName,
	cmd.ConfigFileFlag,
	cmd.ChainConfigFileFlag,
}

func init() {
	appFlags = cmd.WrapFlags(append(appFlags, featureconfig.ValidatorFlags...))
}

func main() {
	app := cli.App{}
	app.Name = "validator"
	app.Usage = `launches an Ethereum Serenity validator client that interacts with a beacon chain,
				 starts proposer services, shardp2p connections, and more`
	app.Version = version.GetVersion()
	app.Action = startNode
	app.Commands = []*cli.Command{
		{
			Name:     "accounts",
			Category: "accounts",
			Usage:    "defines useful functions for interacting with the validator client's account",
			Subcommands: []*cli.Command{
				{
					Name: "create",
					Description: `creates a new validator account keystore containing private keys for Ethereum Serenity -
this command outputs a deposit data string which can be used to deposit Ether into the ETH1.0 deposit
contract in order to activate the validator client`,
					Flags: []cli.Flag{
						flags.KeystorePathFlag,
						flags.PasswordFlag,
					},
					Action: func(ctx *cli.Context) error {
						featureconfig.ConfigureValidator(ctx)
						if featureconfig.Get().MinimalConfig {
							log.Warn("Using Minimal Config")
							params.UseMinimalConfig()
						}

						if keystoreDir, _, err := accounts.CreateValidatorAccount(ctx.String(flags.KeystorePathFlag.Name), ctx.String(flags.PasswordFlag.Name)); err != nil {
							log.WithError(err).Fatalf("Could not create validator at path: %s", keystoreDir)
						}
						return nil
					},
				},
				{
					Name:        "keys",
					Description: `lists the private keys for 'keystore' keymanager keys`,
					Flags: []cli.Flag{
						flags.KeystorePathFlag,
						flags.PasswordFlag,
					},
					Action: func(ctx *cli.Context) error {
						if ctx.String(flags.KeystorePathFlag.Name) == "" {
							log.Fatalf("%s is required", flags.KeystorePathFlag.Name)
						}
						if ctx.String(flags.PasswordFlag.Name) == "" {
							log.Fatalf("%s is required", flags.PasswordFlag.Name)
						}
						keystores, err := accounts.DecryptKeysFromKeystore(ctx.String(flags.KeystorePathFlag.Name), ctx.String(flags.PasswordFlag.Name))
						if err != nil {
							log.WithError(err).Fatalf("Failed to decrypt keystore keys at path %s", ctx.String(flags.KeystorePathFlag.Name))
						}
						for _, v := range keystores {
							fmt.Printf("Public key: %#x private key: %#x\n", v.PublicKey.Marshal(), v.SecretKey.Marshal())
						}
						return nil
					},
				},
			},
		},
	}
	app.Flags = appFlags

	app.Before = func(ctx *cli.Context) error {
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
			// the colors are ANSI codes and seen as Gibberish in the log files.
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

		runtime.GOMAXPROCS(runtime.NumCPU())
		return debug.Setup(ctx)
	}

	app.After = func(ctx *cli.Context) error {
		debug.Exit(ctx)
		return nil
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
