package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"syscall"

	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/logutil"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/node"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	_ "go.uber.org/automaxprocs"
	"golang.org/x/crypto/ssh/terminal"
)

var log = logrus.WithField("prefix", "main")

type unencryptedKeysContainer struct {
	Keys []*unencryptedKeys `json:"keys"`
}

type unencryptedKeys struct {
	ValidatorKey  []byte `json:"validator_key"`
	WithdrawalKey []byte `json:"withdrawal_key"`
}

func startNode(ctx *cli.Context) error {
	// Unsafe start from plain text keys.
	if unencryptedKeys := ctx.String(flags.UnencryptedKeysFlag.Name); unencryptedKeys != "" {
		keys, err := loadUnencryptedKeys(unencryptedKeys)
		if err != nil {
			return err
		}
		validatorClient, err := node.NewValidatorClient(ctx, keys)
		if err != nil {
			return err
		}

		validatorClient.Start()
		return nil
	}

	// Interop start from generated keys.
	if numValidatorKeys := ctx.GlobalUint64(flags.InteropNumValidators.Name); numValidatorKeys > 0 {
		keys, err := interopValidatorKeys(ctx.GlobalUint64(flags.InteropStartIndex.Name), numValidatorKeys)
		if err != nil {
			return err
		}
		validatorClient, err := node.NewValidatorClient(ctx, keys)
		if err != nil {
			return err
		}

		validatorClient.Start()
		return nil
	}

	// Normal production key start.
	keystoreDirectory := ctx.String(flags.KeystorePathFlag.Name)
	keystorePassword := ctx.String(flags.PasswordFlag.Name)

	exists, err := accounts.Exists(keystoreDirectory)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		// If an account does not exist, we create a new one and start the node.
		keystoreDirectory, keystorePassword, err = createValidatorAccount(ctx)
		if err != nil {
			log.Fatalf("Could not create validator account: %v", err)
		}
	} else {
		if keystorePassword == "" {
			log.Info("Enter your validator account password:")
			bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Fatalf("Could not read account password: %v", err)
			}
			text := string(bytePassword)
			keystorePassword = strings.Replace(text, "\n", "", -1)
		}

		if err := accounts.VerifyAccountNotExists(keystoreDirectory, keystorePassword); err == nil {
			log.Info("No account found, creating new validator account...")
		}
	}

	verbosity := ctx.GlobalString(cmd.VerbosityFlag.Name)
	level, err := logrus.ParseLevel(verbosity)
	if err != nil {
		return err
	}
	logrus.SetLevel(level)

	validatorKeys, err := accounts.DecryptKeysFromKeystore(keystoreDirectory, keystorePassword)
	if err != nil {
		log.Fatal(err)
	}

	validatorClient, err := node.NewValidatorClient(ctx, validatorKeys)
	if err != nil {
		return err
	}

	validatorClient.Start()
	return nil
}

func createValidatorAccount(ctx *cli.Context) (string, string, error) {
	keystoreDirectory := ctx.String(flags.KeystorePathFlag.Name)
	keystorePassword := ctx.String(flags.PasswordFlag.Name)
	if keystorePassword == "" {
		reader := bufio.NewReader(os.Stdin)
		logrus.Info("Create a new validator account for eth2")
		log.Info("Enter a password:")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Could not read account password: %v", err)
		}
		text := string(bytePassword)
		keystorePassword = strings.Replace(text, "\n", "", -1)
		log.Infof("Keystore path to save your private keys (leave blank for default %s):", keystoreDirectory)
		text, err = reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		text = strings.Replace(text, "\n", "", -1)
		if text != "" {
			keystoreDirectory = text
		}
	}

	if err := accounts.NewValidatorAccount(keystoreDirectory, keystorePassword); err != nil {
		return "", "", errors.Wrapf(err, "could not initialize validator account")
	}
	return keystoreDirectory, keystorePassword, nil
}

func parseUnencryptedKeysFile(r io.Reader) ([][]byte, [][]byte, error) {
	encoded, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	var ctnr *unencryptedKeysContainer
	if err := json.Unmarshal(encoded, &ctnr); err != nil {
		return nil, nil, err
	}
	validatorKeys := make([][]byte, 0)
	withdrawalKeys := make([][]byte, 0)
	for _, item := range ctnr.Keys {
		validatorKeys = append(validatorKeys, item.ValidatorKey)
		withdrawalKeys = append(withdrawalKeys, item.WithdrawalKey)
	}
	return validatorKeys, withdrawalKeys, nil
}

var appFlags = []cli.Flag{
	flags.NoCustomConfigFlag,
	flags.BeaconRPCProviderFlag,
	flags.CertFlag,
	flags.KeystorePathFlag,
	flags.PasswordFlag,
	flags.DisablePenaltyRewardLogFlag,
	flags.UnencryptedKeysFlag,
	flags.InteropStartIndex,
	flags.InteropNumValidators,
	cmd.VerbosityFlag,
	cmd.DataDirFlag,
	cmd.EnableTracingFlag,
	cmd.TracingProcessNameFlag,
	cmd.TracingEndpointFlag,
	cmd.TraceSampleFractionFlag,
	cmd.BootstrapNode,
	cmd.MonitoringPortFlag,
	cmd.LogFormat,
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
	appFlags = append(appFlags, featureconfig.ValidatorFlags...)
}

func main() {
	app := cli.NewApp()
	app.Name = "validator"
	app.Usage = `launches an Ethereum Serenity validator client that interacts with a beacon chain,
				 starts proposer services, shardp2p connections, and more`
	app.Version = version.GetVersion()
	app.Action = startNode
	app.Commands = []cli.Command{
		{
			Name:     "accounts",
			Category: "accounts",
			Usage:    "defines useful functions for interacting with the validator client's account",
			Subcommands: cli.Commands{
				cli.Command{
					Name: "create",
					Description: `creates a new validator account keystore containing private keys for Ethereum Serenity -
this command outputs a deposit data string which can be used to deposit Ether into the ETH1.0 deposit
contract in order to activate the validator client`,
					Flags: []cli.Flag{
						flags.KeystorePathFlag,
						flags.PasswordFlag,
					},
					Action: func(ctx *cli.Context) {
						if keystoreDir, _, err := createValidatorAccount(ctx); err != nil {
							log.Fatalf("Could not create validator at path: %s", keystoreDir)
						}
					},
				},
			},
		},
	}
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
