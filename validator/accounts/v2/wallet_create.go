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
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if err != nil && !errors.Is(err, ErrNoWalletFound) {
		log.Fatalf("Could not parse wallet directory: %v", err)
	}
	// Check if the user has a wallet at the specified path.
	// If a user does not have a wallet, we instantiate one
	// based on specified options.
	walletExists, err := hasDir(walletDir)
	if err != nil {
		log.Fatal(err)
	}
	if walletExists {
		log.Fatal(
			"You already have a wallet at the specified path. You can " +
				"edit your wallet configuration by running ./prysm.sh validator wallet-v2 edit",
		)
	}
	// Determine the desired keymanager kind for the wallet from user input.
	keymanagerKind, err := inputKeymanagerKind(cliCtx)
	if err != nil {
		log.Fatalf("Could not select keymanager kind: %v", err)
	}
	switch keymanagerKind {
	case v2keymanager.Direct:
		if err = initializeDirectWallet(cliCtx, walletDir); err != nil {
			log.Fatalf("Could not initialize wallet with direct keymanager: %v", err)
		}
		log.WithField("wallet-path", walletDir).Infof(
			"Successfully created wallet with on-disk keymanager configuration. " +
				"Make a new validator account with ./prysm.sh validator accounts-2 new",
		)
	case v2keymanager.Derived:
		if err = initializeDerivedWallet(cliCtx, walletDir); err != nil {
			log.Fatalf("Could not initialize wallet with direct keymanager: %v", err)
		}
		log.WithField("wallet-path", walletDir).Infof(
			"Successfully created HD wallet and saved configuration to disk. " +
				"Make a new validator account with ./prysm.sh validator accounts-2 new",
		)
	case v2keymanager.Remote:
		if err = initializeRemoteSignerWallet(cliCtx, walletDir); err != nil {
			log.Fatalf("Could not initialize wallet with remote keymanager: %v", err)
		}
		log.WithField("wallet-path", walletDir).Infof(
			"Successfully created wallet with remote keymanager configuration",
		)
	default:
		log.Fatalf("Keymanager type %s is not supported", keymanagerKind)
	}
	return nil
}

func initializeDirectWallet(cliCtx *cli.Context, walletDir string) error {
	passwordsDirPath := inputPasswordsDirectory(cliCtx)
	walletConfig := &WalletConfig{
		PasswordsDir:      passwordsDirPath,
		WalletDir:         walletDir,
		KeymanagerKind:    v2keymanager.Direct,
		CanUnlockAccounts: true,
	}
	ctx := context.Background()
	wallet, err := NewWallet(ctx, walletConfig)
	if err != nil {
		return errors.Wrap(err, "could not create new wallet")
	}
	keymanagerConfig, err := direct.MarshalConfigFile(ctx, direct.DefaultConfig())
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	return nil
}

func initializeDerivedWallet(cliCtx *cli.Context, walletDir string) error {
	passwordsDirPath := inputPasswordsDirectory(cliCtx)
	walletConfig := &WalletConfig{
		PasswordsDir:      passwordsDirPath,
		WalletDir:         walletDir,
		KeymanagerKind:    v2keymanager.Derived,
		CanUnlockAccounts: true,
	}
	ctx := context.Background()
	walletPassword, err := inputNewWalletPassword()
	if err != nil {
		return err
	}
	seedConfig, err := derived.InitializeWalletSeedFile(ctx, walletPassword)
	if err != nil {
		return err
	}
	seedConfigFile, err := derived.MarshalEncryptedSeedFile(ctx, seedConfig)
	if err != nil {
		return err
	}
	wallet, err := NewWallet(ctx, walletConfig)
	if err != nil {
		return errors.Wrap(err, "could not create new wallet")
	}
	keymanagerConfig, err := derived.MarshalConfigFile(ctx, derived.DefaultConfig())
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	if err := wallet.WriteEncryptedSeedToDisk(ctx, seedConfigFile); err != nil {
		return errors.Wrap(err, "could not write encrypted wallet seed config to disk")
	}
	return nil
}

func initializeRemoteSignerWallet(cliCtx *cli.Context, walletDir string) error {
	conf, err := inputRemoteKeymanagerConfig(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not input remote keymanager config")
	}
	ctx := context.Background()
	keymanagerConfig, err := remote.MarshalConfigFile(ctx, conf)
	if err != nil {
		return errors.Wrap(err, "could not marshal config file")
	}
	walletConfig := &WalletConfig{
		WalletDir:      walletDir,
		KeymanagerKind: v2keymanager.Remote,
	}
	wallet, err := NewWallet(ctx, walletConfig)
	if err != nil {
		return errors.Wrap(err, "could not create new wallet")
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
