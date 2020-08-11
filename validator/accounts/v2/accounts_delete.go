package v2

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/urfave/cli/v2"
)

// DeleteAccount deletes the accounts that the user requests to be deleted from the wallet.
func DeleteAccount(cliCtx *cli.Context) error {
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx)
	if errors.Is(err, ErrNoWalletFound) {
		return errors.Wrap(err, "no wallet found at path, create a new wallet with wallet-v2 create")
	} else if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	skipMnemonicConfirm := cliCtx.Bool(flags.SkipMnemonicConfirmFlag.Name)
	keymanager, err := wallet.InitializeKeymanager(cliCtx, skipMnemonicConfirm)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	allAccounts, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return err
	}
	if len(allAccounts) == 0 {
		return errors.New("wallet is empty, no accounts to delete")
	}
	allAccountStrs := make([]string, len(allAccounts))
	for i, account := range allAccounts {
		allAccountStrs[i] = fmt.Sprintf("%#x", bytesutil.FromBytes48(account))
	}
	accounts, err := selectAccounts(cliCtx, selectAccountsDeletePromptText, allAccountStrs)
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return errors.New("no accounts selected to delete")
	}

	formattedPubKeys := make([]string, len(accounts))
	for i, pubKey := range accounts {
		keyBytes, err := hex.DecodeString(pubKey[2:])
		if err != nil {
			return errors.Wrap(err, "could not decode hex string")
		}
		formattedPubKeys[i] = fmt.Sprintf("%#x", bytesutil.Trunc(keyBytes))
	}
	allAccountStr := strings.Join(formattedPubKeys, ", ")

	if len(accounts) == 1 {
		promptText := "Are you sure you want to delete 1 account? (%s) Y/y"
		_, err = promptutil.ValidatePrompt(fmt.Sprintf(promptText, au.BrightGreen(formattedPubKeys[0])), promptutil.ValidateConfirmation)
		if err != nil {
			return err
		}
	} else {
		promptText := "Are you sure you want to delete %d accounts? (%s) Y/y"
		if len(accounts) == len(allAccounts) {
			promptText = fmt.Sprintf("Are you sure you want to delete all accounts? (%s)", au.BrightGreen(allAccountStr))
		} else {
			promptText = fmt.Sprintf(promptText, len(accounts), au.BrightGreen(allAccountStr))
		}
		_, err = promptutil.ValidatePrompt(promptText, promptutil.ValidateConfirmation)
		if err != nil {
			return err
		}
	}
	switch wallet.KeymanagerKind() {
	case v2keymanager.Remote:
		return errors.New("cannot delete accounts for a remote keymanager")
	case v2keymanager.Direct:
		km, ok := keymanager.(*direct.Keymanager)
		if !ok {
			return errors.New("not a direct keymanager")
		}
		if len(accounts) == 1 {
			log.Info("Deleting account...")
		} else {
			log.Info("Deleting accounts...")
		}
		// Delete the requested account's keystore.
		for _, pubKey := range accounts {
			pubKeyBytes, err := hex.DecodeString(pubKey[2:])
			if err != nil {
				return errors.Wrapf(err, "could not decode public key %s", pubKey)
			}
			if err := km.DeleteAccount(ctx, pubKeyBytes); err != nil {
				return errors.Wrap(err, "could not create account in wallet")
			}
		}
	case v2keymanager.Derived:
		return errors.New("cannot delete accounts for a derived keymanager")
	default:
		return fmt.Errorf("keymanager kind %s not supported", wallet.KeymanagerKind())
	}
	log.WithField("publicKeys", allAccountStr).Info("Accounts deleted")
	return nil
}
