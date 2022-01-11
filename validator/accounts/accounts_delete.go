package accounts

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/io/prompt"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/userprompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/urfave/cli/v2"
)

// DeleteAccountCli deletes the accounts that the user requests to be deleted from the wallet.
// This function uses the CLI to extract necessary values.
func DeleteAccountCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, wallet.ErrNoWalletFound
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	kManager, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	if err != nil {
		return errors.Wrap(err, ErrCouldNotInitializeKeymanager)
	}
	validatingPublicKeys, err := kManager.FetchValidatingPublicKeys(cliCtx.Context)
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
		userprompt.SelectAccountsDeletePromptText,
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
			resp, err := prompt.ValidatePrompt(
				os.Stdin, fmt.Sprintf(promptText, au.BrightGreen(formattedPubKeys[0])), prompt.ValidateYesOrNo,
			)
			if err != nil {
				return err
			}
			if strings.EqualFold(resp, "n") {
				return nil
			}
		} else {
			promptText := "Are you sure you want to delete %d accounts? (%s) Y/N"
			if len(filteredPubKeys) == len(validatingPublicKeys) {
				promptText = fmt.Sprintf("Are you sure you want to delete all accounts? Y/N (%s)", au.BrightGreen(allAccountStr))
			} else {
				promptText = fmt.Sprintf(promptText, len(filteredPubKeys), au.BrightGreen(allAccountStr))
			}
			resp, err := prompt.ValidatePrompt(os.Stdin, promptText, prompt.ValidateYesOrNo)
			if err != nil {
				return err
			}
			if strings.EqualFold(resp, "n") {
				return nil
			}
		}
	}
	if err := DeleteAccount(cliCtx.Context, &Config{
		Wallet:           w,
		Keymanager:       kManager,
		DeletePublicKeys: rawPublicKeys,
	}); err != nil {
		return err
	}
	log.WithField("publicKeys", allAccountStr).Warn(
		"Attempted to delete accounts. IMPORTANT: please run `validator accounts list` to ensure " +
			"the public keys are indeed deleted. If they are still there, please file an issue at " +
			"https://github.com/prysmaticlabs/prysm/issues/new")
	return nil
}

// DeleteAccount deletes the accounts that the user requests to be deleted from the wallet.
func DeleteAccount(ctx context.Context, cfg *Config) error {
	deleter, ok := cfg.Keymanager.(keymanager.Deleter)
	if !ok {
		return errors.New("keymanager does not implement Deleter interface")
	}
	if len(cfg.DeletePublicKeys) == 1 {
		log.Info("Deleting account...")
	} else {
		log.Info("Deleting accounts...")
	}
	statuses, err := deleter.DeleteKeystores(ctx, cfg.DeletePublicKeys)
	if err != nil {
		return errors.Wrap(err, "could not delete accounts")
	}
	for i, status := range statuses {
		switch status.Status {
		case ethpbservice.DeletedKeystoreStatus_ERROR:
			log.Errorf("Error deleting key %#x: %s", bytesutil.Trunc(cfg.DeletePublicKeys[i]), status.Message)
		case ethpbservice.DeletedKeystoreStatus_NOT_ACTIVE:
			log.Warnf("Duplicate key %#x found in delete request", bytesutil.Trunc(cfg.DeletePublicKeys[i]))
		case ethpbservice.DeletedKeystoreStatus_NOT_FOUND:
			log.Warnf("Could not find keystore for %#x", bytesutil.Trunc(cfg.DeletePublicKeys[i]))
		}
	}
	return nil
}
