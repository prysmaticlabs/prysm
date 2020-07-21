package v2

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

// EditWalletConfiguration for a user's on-disk wallet, being able to change
// things such as remote gRPC credentials for remote signing, derivation paths
// for HD wallets, and more.
func EditWalletConfiguration(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if errors.Is(err, ErrNoWalletFound) {
		log.Fatal("No wallet found, create a new one with ./prysm.sh validator wallet-v2 create")
	} else if err != nil {
		log.Fatal("Could not parse wallet directory")
	}
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx, &WalletConfig{
		CanUnlockAccounts: false,
		WalletDir:         walletDir,
	})
	if err != nil {
		log.Fatalf("Could not open wallet: %v", err)
	}
	switch wallet.KeymanagerKind() {
	case v2keymanager.Direct:
		log.Fatal("No configuration options available to edit for direct keymanager")
	case v2keymanager.Derived:
		log.Fatal("Derived keymanager is not yet supported")
	case v2keymanager.Remote:
		enc, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
		if err != nil {
			log.Fatalf("Could not read: %v", err)
		}
		cfg, err := remote.UnmarshalConfigFile(enc)
		if err != nil {
			log.Fatalf("Could not unmarshal: %v", err)
		}
		log.Infof("Current configuration")
		fmt.Printf("%s\n", cfg)
		newCfg, err := inputRemoteKeymanagerConfig(cliCtx)
		if err != nil {
			log.Fatal(err)
		}
		encodedCfg, err := remote.MarshalConfigFile(ctx, newCfg)
		if err != nil {
			log.Fatal(err)
		}
		if err := wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Keymanager type %s is not supported", wallet.KeymanagerKind())
	}
	return nil
}
