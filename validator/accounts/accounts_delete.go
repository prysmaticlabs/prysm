package accounts

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/urfave/cli/v2"
)

// DeleteAccountCli deletes the accounts that the user requests to be deleted from the wallet.
// This function uses the CLI to extract necessary values.
func DeleteAccountCli(cliCtx *cli.Context) error {
	accountCfg, err := SelectPublicKeysFromAccount(cliCtx, flags.DeletePublicKeysFlag)
	if err != nil {
		return err
	}
	formattedPubKeys := make([]string, len(accountCfg.PublicKeys))
	for i, pk := range accountCfg.PublicKeys {
		formattedPubKeys[i] = fmt.Sprintf("%#x", bytesutil.Trunc(pk))
	}
	allAccountStr := strings.Join(formattedPubKeys, ", ")
	if !cliCtx.IsSet(flags.DeletePublicKeysFlag.Name) {
		if len(accountCfg.PublicKeys) == 1 {
			promptText := "Are you sure you want to delete 1 account? (%s) Y/N"
			resp, err := promptutil.ValidatePrompt(
				os.Stdin, fmt.Sprintf(promptText, au.BrightGreen(formattedPubKeys[0])), promptutil.ValidateYesOrNo,
			)
			if err != nil {
				return err
			}
			if strings.ToLower(resp) == "n" {
				return nil
			}
		} else {
			promptText := "Are you sure you want to delete %d accounts? (%s) Y/N"
			validatingPublicKeys, err := accountCfg.Keymanager.FetchValidatingPublicKeys(cliCtx.Context)
			if err != nil {
				return err
			}
			if len(accountCfg.PublicKeys) == len(validatingPublicKeys) {
				promptText = fmt.Sprintf("Are you sure you want to delete all accounts? Y/N (%s)", au.BrightGreen(allAccountStr))
			} else {
				promptText = fmt.Sprintf(promptText, len(accountCfg.PublicKeys), au.BrightGreen(allAccountStr))
			}
			resp, err := promptutil.ValidatePrompt(os.Stdin, promptText, promptutil.ValidateYesOrNo)
			if err != nil {
				return err
			}
			if strings.ToLower(resp) == "n" {
				return nil
			}
		}
	}
	if err := DeleteAccount(cliCtx.Context, accountCfg); err != nil {
		return err
	}
	log.WithField("publicKeys", allAccountStr).Info("Accounts deleted")
	return nil
}

// DeleteAccount deletes the accounts that the user requests to be deleted from the wallet.
func DeleteAccount(ctx context.Context, cfg *AccountConfig) error {
	switch cfg.Wallet.KeymanagerKind() {
	case keymanager.Remote:
		return errors.New("cannot delete accounts for a remote keymanager")
	case keymanager.Imported:
		km, ok := cfg.Keymanager.(*imported.Keymanager)
		if !ok {
			return errors.New("not a imported keymanager")
		}
		if len(cfg.PublicKeys) == 1 {
			log.Info("Deleting account...")
		} else {
			log.Info("Deleting accounts...")
		}
		if err := km.DeleteAccounts(ctx, cfg.PublicKeys); err != nil {
			return errors.Wrap(err, "could not delete accounts")
		}
	case keymanager.Derived:
		return errors.New("cannot delete accounts for a derived keymanager")
	default:
		return fmt.Errorf("keymanager kind %s not supported", cfg.Wallet.KeymanagerKind())
	}
	return nil
}
