package v2

import (
	"context"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
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
	if err != nil {
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
				"edit your wallet configuration by running ./prysm validator wallet-v2 edit",
		)
	}
	// Determine the desired keymanager kind for the wallet from user input.
	keymanagerKind, err := inputKeymanagerKind(cliCtx)
	if err != nil {
		log.Fatalf("Could not select keymanager kind: %v", err)
	}
	switch keymanagerKind {
	case v2keymanager.Direct:
		if err := initializeDirectWallet(cliCtx, walletDir); err != nil {
			log.Fatalf("Could not initialize wallet with direct keymanager: %v", err)
		}
		log.Infof(
			"Successfully created wallet with on-disk keymanager configuration. " +
				"Make a new validator account with ./prysm validator wallet-v2 accounts new",
		)
	case v2keymanager.Derived:
		log.Fatal("Derived keymanager is not yet supported")
	case v2keymanager.Remote:
		if err := initializeRemoteSignerWallet(cliCtx); err != nil {
			log.Fatalf("Could not initialize wallet with remote keymanager: %v", err)
		}
		log.Infof(
			"Successfully created wallet with remote keymanager configuration",
		)
	default:
		log.Fatalf("Keymanager type %s is not supported", keymanagerKind.String())
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
	keymanager, err := direct.NewKeymanager(ctx, wallet, direct.DefaultConfig())
	if err != nil {
		return errors.Wrap(err, "could not create new direct keymanager")
	}
	keymanagerConfig, err := keymanager.MarshalConfigFile(ctx)
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	return wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig)
}

func initializeRemoteSignerWallet(cliCtx *cli.Context, walletDir string) error {
	conf, err := inputRemoteKeymanagerConfig(cliCtx)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	keymanager, err := remote.NewKeymanager(
		ctx,
		0, /* max grpc msg size, not relevant */
		conf,
	)
	if err != nil {
		log.Fatal(err)
	}
	keymanagerConfig, err := keymanager.MarshalConfigFile(ctx)
	if err != nil {
		log.Fatal(err)
	}
	walletConfig := &WalletConfig{
		WalletDir:      walletDir,
		KeymanagerKind: v2keymanager.Remote,
	}
	wallet, err := NewWallet(ctx, walletConfig)
	if err != nil {
		return errors.Wrap(err, "could not create new wallet")
	}
	return wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig)
}

func inputRemoteKeymanagerConfig(cliCtx *cli.Context) (*remote.Config, error) {
	prompt := promptui.Prompt{
		Label: "Remote gRPC address (such as host.example.com:4000)",
		Validate: func(input string) error {
			// TODO: Validate if it is a valid address.
			if input == "" {
				return errors.New("cannot be empty")
			}
			return nil
		},
	}
	remoteAddr, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	prompt = promptui.Prompt{
		Label: "Path to TLS crt (such as /path/to/client.crt)",
		Validate: func(input string) error {
			// TODO: Validate if it is a valid address.
			if input == "" {
				return errors.New("cannot be empty")
			}
			return nil
		},
	}
	clientCrtPath, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	prompt = promptui.Prompt{
		Label: "Path to TLS key (such as /path/to/client.key)",
		Validate: func(input string) error {
			// TODO: Validate if it is a valid path.
			if input == "" {
				return errors.New("cannot be empty")
			}
			return nil
		},
	}
	clientKeyPath, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	prompt = promptui.Prompt{
		Label: "(Optional) Path to certificate authority (CA) crt (such as /path/to/ca.crt)",
		Validate: func(input string) error {
			// TODO: Validate if it is a valid path.
			if input == "" {
				return errors.New("cannot be empty")
			}
			return nil
		},
	}
	caCrtPath, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return &remote.Config{
		RemoteCertificate: &remote.CertificateConfig{
			ClientCertPath: clientCrtPath,
			ClientKeyPath:  clientKeyPath,
			CACertPath:     caCrtPath,
		},
		RemoteAddr: remoteAddr,
	}, nil
}
