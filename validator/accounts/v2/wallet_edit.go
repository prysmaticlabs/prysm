package v2

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
)

// EditWalletConfiguration for a user's on-disk wallet, being able to change
// things such as remote gRPC credentials for remote signing, derivation paths
// for HD wallets, and more.
func EditWalletConfiguration(cliCtx *cli.Context) error {
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	switch wallet.KeymanagerKind() {
	case v2keymanager.Direct:
		return errors.New("not possible to edit direct keymanager configuration")
	case v2keymanager.Derived:
		return errors.New("derived keymanager is not yet supported")
	case v2keymanager.Remote:
		enc, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
		if err != nil {
			return errors.Wrap(err, "could not read config")
		}
		cfg, err := remote.UnmarshalConfigFile(enc)
		if err != nil {
			return errors.Wrap(err, "could not unmarshal config")
		}
		log.Info("Current configuration")
		// Prints the current configuration to stdout.
		fmt.Println(cfg)
		newCfg, err := inputRemoteKeymanagerConfig(cliCtx)
		if err != nil {
			return errors.Wrap(err, "could not get keymanager config")
		}
		encodedCfg, err := remote.MarshalConfigFile(ctx, newCfg)
		if err != nil {
			return errors.Wrap(err, "could not marshal config file")
		}
		if err := wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg); err != nil {
			return errors.Wrap(err, "could not write config to disk")
		}
	default:
		return fmt.Errorf("keymanager type %s is not supported", wallet.KeymanagerKind())
	}
	return nil
}
