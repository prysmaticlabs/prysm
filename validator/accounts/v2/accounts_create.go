package v2

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "accounts-v2")

// CreateAccountConfig to run the create account function.
type CreateAccountConfig struct {
	Wallet      *wallet.Wallet
	NumAccounts int64
}

// CreateAccountCli creates a new validator account from user input by opening
// a wallet from the user's specified path. This uses the CLI to extract information
// to perform account creation.
func CreateAccountCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, CreateAndSaveWalletCli)
	if err != nil {
		return err
	}
	numAccounts := cliCtx.Int64(flags.NumAccountsFlag.Name)
	log.Info("Creating a new account...")
	return CreateAccount(cliCtx.Context, &CreateAccountConfig{
		Wallet:      w,
		NumAccounts: numAccounts,
	})
}

// CreateAccount creates a new validator account from user input by opening
// a wallet from the user's specified path.
func CreateAccount(ctx context.Context, cfg *CreateAccountConfig) error {
	keymanager, err := cfg.Wallet.InitializeKeymanager(ctx, false /* skip mnemonic confirm */)
	if err != nil && strings.Contains(err.Error(), "invalid checksum") {
		return errors.New("wrong wallet password entered")
	}
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	switch cfg.Wallet.KeymanagerKind() {
	case v2keymanager.Remote:
		return errors.New("cannot create a new account for a remote keymanager")
	case v2keymanager.Direct:
		km, ok := keymanager.(*direct.Keymanager)
		if !ok {
			return errors.New("not a direct keymanager")
		}
		// Create a new validator account using the specified keymanager.
		if _, err := km.CreateAccount(ctx); err != nil {
			return errors.Wrap(err, "could not create account in wallet")
		}
	case v2keymanager.Derived:
		km, ok := keymanager.(*derived.Keymanager)
		if !ok {
			return errors.New("not a derived keymanager")
		}
		startNum := km.NextAccountNumber(ctx)
		if cfg.NumAccounts == 1 {
			if _, err := km.CreateAccount(ctx, true /*logAccountInfo*/); err != nil {
				return errors.Wrap(err, "could not create account in wallet")
			}
		} else {
			for i := 0; i < int(cfg.NumAccounts); i++ {
				if _, err := km.CreateAccount(ctx, false /*logAccountInfo*/); err != nil {
					return errors.Wrap(err, "could not create account in wallet")
				}
			}
			log.Infof(
				"Successfully created %d accounts. Please use accounts-v2 list to view details for accounts %d through %d",
				cfg.NumAccounts,
				startNum,
				startNum+uint64(cfg.NumAccounts)-1,
			)
		}
	default:
		return fmt.Errorf("keymanager kind %s not supported", cfg.Wallet.KeymanagerKind())
	}
	return nil
}
