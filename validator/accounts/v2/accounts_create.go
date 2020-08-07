package v2

import (
	"context"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "accounts-v2")

// CreateAccount creates a new validator account from user input by opening
// a wallet from the user's specified path.
func CreateAccount(cliCtx *cli.Context) error {
	ctx := context.Background()
	wallet, err := createOrOpenWallet(cliCtx, CreateWallet)
	if err != nil {
		return err
	}
	skipMnemonicConfirm := cliCtx.Bool(flags.SkipMnemonicConfirmFlag.Name)
	keymanager, err := wallet.InitializeKeymanager(cliCtx, skipMnemonicConfirm)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	log.Info("Creating a new account...")
	switch wallet.KeymanagerKind() {
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
		numAccounts := cliCtx.Int64(flags.NumAccountsFlag.Name)
		if numAccounts == 1 {
			if _, err := km.CreateAccount(ctx, true /*logAccountInfo*/); err != nil {
				return errors.Wrap(err, "could not create account in wallet")
			}
		} else {
			for i := 0; i < int(numAccounts); i++ {
				if _, err := km.CreateAccount(ctx, false /*logAccountInfo*/); err != nil {
					return errors.Wrap(err, "could not create account in wallet")
				}
			}
			log.Infof("Successfully created %d accounts. Please use accounts-v2 list to view details for accounts %d through %d.", numAccounts, startNum, startNum+uint64(numAccounts)-1)
		}
	default:
		return fmt.Errorf("keymanager kind %s not supported", wallet.KeymanagerKind())
	}
	return nil
}

func inputKeymanagerKind(cliCtx *cli.Context) (v2keymanager.Kind, error) {
	if cliCtx.IsSet(flags.KeymanagerKindFlag.Name) {
		return v2keymanager.ParseKind(cliCtx.String(flags.KeymanagerKindFlag.Name))
	}
	promptSelect := promptui.Select{
		Label: "Select a type of wallet",
		Items: []string{
			keymanagerKindSelections[v2keymanager.Derived],
			keymanagerKindSelections[v2keymanager.Direct],
			keymanagerKindSelections[v2keymanager.Remote],
		},
	}
	selection, _, err := promptSelect.Run()
	if err != nil {
		return v2keymanager.Direct, fmt.Errorf("could not select wallet type: %v", formatPromptError(err))
	}
	return v2keymanager.Kind(selection), nil
}
