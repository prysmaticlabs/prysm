package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/node"
	"github.com/prysmaticlabs/prysm/validator/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/crypto/ssh/terminal"
)

func startNode(ctx *cli.Context) error {
	keystoreDirectory := ctx.String(types.KeystorePathFlag.Name)
	keystorePassword := ctx.String(types.PasswordFlag.Name)

	exists, err := accounts.Exists(keystoreDirectory)
	if err != nil {
		logrus.Fatal(err)
	}
	if !exists {
		// If an account does not exist, we create a new one and start the node.
		keystoreDirectory, keystorePassword, err = createValidatorAccount(ctx)
		if err != nil {
			logrus.Fatalf("Could not create validator account: %v", err)
		}
	} else {
		if keystorePassword == "" {
			logrus.Info("Enter your validator account password:")
			bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				logrus.Fatalf("Could not read account password: %v", err)
			}
			text := string(bytePassword)
			keystorePassword = strings.Replace(text, "\n", "", -1)
		}

		if err := accounts.VerifyAccountNotExists(keystoreDirectory, keystorePassword); err == nil {
			logrus.Info("No account found, creating new validator account...")
		}
	}

	verbosity := ctx.GlobalString(cmd.VerbosityFlag.Name)
	level, err := logrus.ParseLevel(verbosity)
	if err != nil {
		return err
	}
	logrus.SetLevel(level)

	validatorClient, err := node.NewValidatorClient(ctx, keystorePassword)
	if err != nil {
		return err
	}

	validatorClient.Start()
	return nil
}

func createValidatorAccount(ctx *cli.Context) (string, string, error) {
	keystoreDirectory := ctx.String(types.KeystorePathFlag.Name)
	keystorePassword := ctx.String(types.PasswordFlag.Name)
	if keystorePassword == "" {
		reader := bufio.NewReader(os.Stdin)
		logrus.Info("Create a new validator account for eth2")
		logrus.Info("Enter a password:")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			logrus.Fatalf("Could not read account password: %v", err)
		}
		text := string(bytePassword)
		keystorePassword = strings.Replace(text, "\n", "", -1)
		logrus.Infof("Keystore path to save your private keys (leave blank for default %s):", keystoreDirectory)
		text, err = reader.ReadString('\n')
		if err != nil {
			logrus.Fatal(err)
		}
		text = strings.Replace(text, "\n", "", -1)
		if text != "" {
			keystoreDirectory = text
		}
	}

	if err := accounts.NewValidatorAccount(keystoreDirectory, keystorePassword); err != nil {
		return "", "", fmt.Errorf("could not initialize validator account: %v", err)
	}
	return keystoreDirectory, keystorePassword, nil
}

func main() {
	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)
	log := logrus.WithField("prefix", "main")
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
						types.KeystorePathFlag,
						types.PasswordFlag,
					},
					Action: func(ctx *cli.Context) {
						if keystoreDir, _, err := createValidatorAccount(ctx); err != nil {
							logrus.Fatalf("Could not create validator at path: %s", keystoreDir)
						}
					},
				},
			},
		},
	}
	app.Flags = []cli.Flag{
		types.NoCustomConfigFlag,
		types.BeaconRPCProviderFlag,
		types.KeystorePathFlag,
		types.PasswordFlag,
		types.DisablePenaltyRewardLogFlag,
		cmd.VerbosityFlag,
		cmd.DataDirFlag,
		cmd.EnableTracingFlag,
		cmd.TracingEndpointFlag,
		cmd.TraceSampleFractionFlag,
		cmd.BootstrapNode,
		cmd.MonitoringPortFlag,
		debug.PProfFlag,
		debug.PProfAddrFlag,
		debug.PProfPortFlag,
		debug.MemProfileRateFlag,
		debug.CPUProfileFlag,
		debug.TraceFlag,
	}

	app.Flags = append(app.Flags, featureconfig.ValidatorFlags...)

	app.Before = func(ctx *cli.Context) error {
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
