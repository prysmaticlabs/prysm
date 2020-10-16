package accounts

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/prompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/urfave/cli/v2"
)

// DeleteAccountConfig specifies parameters to run the delete account function.
type DeleteAccountConfig struct {
	Wallet     *wallet.Wallet
	Keymanager keymanager.IKeymanager
	PublicKeys [][]byte
}

// DeleteAccountCli deletes the accounts that the user requests to be deleted from the wallet.
// This function uses the CLI to extract necessary values.
func DeleteAccountCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, errors.New(
			"no wallet found, nothing to delete",
		)
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	keymanager, err := w.InitializeKeymanager(cliCtx.Context, false /* skip mnemonic confirm */)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	validatingPublicKeys, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
	if err != nil {
		return err
	}
	if len(validatingPublicKeys) == 0 {
		return errors.New("wallet is empty, no accounts to delete")
	}
	// Allow the user to interactively select the accounts to delete or optionally
	// provide them via cli flags as a string of comma-separated, hex strings.
	filteredPubKeys, err := filterPublicKeysFromUserInput(
		cliCtx,
		flags.DeletePublicKeysFlag,
		validatingPublicKeys,
		prompt.SelectAccountsDeletePromptText,
	)
	if err != nil {
		return errors.Wrap(err, "could not filter public keys for deletion")
	}
	rawPublicKeys := make([][]byte, len(filteredPubKeys))
	formattedPubKeys := make([]string, len(filteredPubKeys))
	for i, pk := range filteredPubKeys {
		pubKeyBytes := pk.Marshal()
		rawPublicKeys[i] = pubKeyBytes
		formattedPubKeys[i] = fmt.Sprintf("%#x", bytesutil.Trunc(pubKeyBytes))
	}
	allAccountStr := strings.Join(formattedPubKeys, ", ")
	if !cliCtx.IsSet(flags.DeletePublicKeysFlag.Name) {
		if len(filteredPubKeys) == 1 {
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
			if len(filteredPubKeys) == len(validatingPublicKeys) {
				promptText = fmt.Sprintf("Are you sure you want to delete all accounts? Y/N (%s)", au.BrightGreen(allAccountStr))
			} else {
				promptText = fmt.Sprintf(promptText, len(filteredPubKeys), au.BrightGreen(allAccountStr))
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
	if err := DeleteAccount(cliCtx.Context, &DeleteAccountConfig{
		Wallet:     w,
		Keymanager: keymanager,
		PublicKeys: rawPublicKeys,
	}); err != nil {
		return err
	}
	log.WithField("publicKeys", allAccountStr).Info("Accounts deleted")
	return nil
}

// DeleteAccount deletes the accounts that the user requests to be deleted from the wallet.
func DeleteAccount(ctx context.Context, cfg *DeleteAccountConfig) error {
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
