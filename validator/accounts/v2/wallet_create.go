package v2

import (
	"context"
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

// CreateWallet from user input with a desired keymanager. If a
// wallet already exists in the path, it suggests the user alternatives
// such as how to edit their existing wallet configuration.
func CreateWallet(cliCtx *cli.Context) error {
	w, err := NewWallet(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not check if wallet directory exists")
	}
	switch w.KeymanagerKind() {
	case v2keymanager.Direct:
		if err = createDirectKeymanagerWallet(cliCtx, w); err != nil {
			return errors.Wrap(err, "could not initialize wallet with direct keymanager")
		}
		log.WithField("wallet-path", w.accountsPath).Infof(
			"Successfully created wallet with on-disk keymanager configuration. " +
				"Make a new validator account with ./prysm.sh validator accounts-2 new",
		)
	case v2keymanager.Derived:
		if err = createDerivedKeymanagerWallet(cliCtx, w); err != nil {
			return errors.Wrap(err, "could not initialize wallet with derived keymanager")
		}
		log.WithField("wallet-path", w.accountsPath).Infof(
			"Successfully created HD wallet and saved configuration to disk. " +
				"Make a new validator account with ./prysm.sh validator accounts-2 new",
		)
	case v2keymanager.Remote:
		if err = createRemoteKeymanagerWallet(cliCtx, w); err != nil {
			return errors.Wrap(err, "could not initialize wallet with remote keymanager")
		}
		log.WithField("wallet-path", w.accountsPath).Infof(
			"Successfully created wallet with remote keymanager configuration",
		)
	default:
		return errors.Wrapf(err, "keymanager type %s is not supported", w.KeymanagerKind())
	}
	return nil
}

func createDirectKeymanagerWallet(cliCtx *cli.Context, wallet *Wallet) error {
	keymanagerConfig, err := direct.MarshalConfigFile(context.Background(), direct.DefaultConfig())
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(context.Background(), keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	return nil
}

func createDerivedKeymanagerWallet(cliCtx *cli.Context, wallet *Wallet) error {
	skipMnemonicConfirm := cliCtx.Bool(flags.SkipMnemonicConfirmFlag.Name)
	ctx := context.Background()
	walletPassword, err := inputNewWalletPassword(cliCtx)
	if err != nil {
		return err
	}
	seedConfig, err := derived.InitializeWalletSeedFile(ctx, walletPassword, skipMnemonicConfirm)
	if err != nil {
		return errors.Wrap(err, "could not initialize new wallet seed file")
	}
	seedConfigFile, err := derived.MarshalEncryptedSeedFile(ctx, seedConfig)
	if err != nil {
		return errors.Wrap(err, "could not marshal encrypted wallet seed file")
	}
	keymanagerConfig, err := derived.MarshalConfigFile(ctx, derived.DefaultConfig())
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, seedConfigFile); err != nil {
		return errors.Wrap(err, "could not write encrypted wallet seed config to disk")
	}
	return nil
}

func createRemoteKeymanagerWallet(cliCtx *cli.Context, wallet *Wallet) error {
	conf, err := inputRemoteKeymanagerConfig(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not input remote keymanager config")
	}
	ctx := context.Background()
	keymanagerConfig, err := remote.MarshalConfigFile(ctx, conf)
	if err != nil {
		return errors.Wrap(err, "could not marshal config file")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	return nil
}

func inputRemoteKeymanagerConfig(cliCtx *cli.Context) (*remote.Config, error) {
	addr := cliCtx.String(flags.GrpcRemoteAddressFlag.Name)
	crt := cliCtx.String(flags.RemoteSignerCertPathFlag.Name)
	key := cliCtx.String(flags.RemoteSignerKeyPathFlag.Name)
	ca := cliCtx.String(flags.RemoteSignerCACertPathFlag.Name)
	if addr != "" && crt != "" && key != "" && ca != "" {
		newCfg := &remote.Config{
			RemoteCertificate: &remote.CertificateConfig{
				ClientCertPath: strings.TrimRight(crt, "\r\n"),
				ClientKeyPath:  strings.TrimRight(key, "\r\n"),
				CACertPath:     strings.TrimRight(ca, "\r\n"),
			},
			RemoteAddr: strings.TrimRight(addr, "\r\n"),
		}
		log.Infof("New configuration")
		fmt.Printf("%s\n", newCfg)
		return newCfg, nil
	}
	log.Infof("Input desired configuration")
	prompt := promptui.Prompt{
		Label: "Remote gRPC address (such as host.example.com:4000)",
		Validate: func(input string) error {
			if input == "" {
				return errors.New("remote host address cannot be empty")
			}
			return nil
		},
	}
	remoteAddr, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	prompt = promptui.Prompt{
		Label:    "Path to TLS crt (such as /path/to/client.crt)",
		Validate: validateCertPath,
	}
	clientCrtPath, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	prompt = promptui.Prompt{
		Label:    "Path to TLS key (such as /path/to/client.key)",
		Validate: validateCertPath,
	}
	clientKeyPath, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	prompt = promptui.Prompt{
		Label:    "(Optional) Path to certificate authority (CA) crt (such as /path/to/ca.crt)",
		Validate: validateCertPath,
	}
	caCrtPath, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	newCfg := &remote.Config{
		RemoteCertificate: &remote.CertificateConfig{
			ClientCertPath: strings.TrimRight(clientCrtPath, "\r\n"),
			ClientKeyPath:  strings.TrimRight(clientKeyPath, "\r\n"),
			CACertPath:     strings.TrimRight(caCrtPath, "\r\n"),
		},
		RemoteAddr: strings.TrimRight(remoteAddr, "\r\n"),
	}
	fmt.Printf("%s\n", newCfg)
	return newCfg, nil
}

func validateCertPath(input string) error {
	if input == "" {
		return errors.New("crt path cannot be empty")
	}
	if !fileExists(input) {
		return fmt.Errorf("no crt found at path: %s", input)
	}
	return nil
}
