package v2

import (
	"context"
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
	validatingPublicKeys, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return err
	}
	if len(validatingPublicKeys) == 0 {
		return errors.New("wallet is empty, no accounts to delete")
	}
	// Allow the user to interactively select the accounts to backup or optionally
	// provide them via cli flags as a string of comma-separated, hex strings.
	filteredPubKeys, err := filterPublicKeysFromUserInput(
		cliCtx,
		flags.DeletePublicKeysFlag,
		validatingPublicKeys,
		selectAccountsDeletePromptText,
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
	if len(filteredPubKeys) == 1 {
		promptText := "Are you sure you want to delete 1 account? (%s) Y/N"
		resp, err := promptutil.ValidatePrompt(fmt.Sprintf(promptText, au.BrightGreen(formattedPubKeys[0])), promptutil.ValidateConfirmation)
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
		resp, err := promptutil.ValidatePrompt(promptText, promptutil.ValidateYesOrNo)
		if err != nil {
			return err
		}
		if strings.ToLower(resp) == "n" {
			return nil
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
		if len(filteredPubKeys) == 1 {
			log.Info("Deleting account...")
		} else {
			log.Info("Deleting accounts...")
		}
		if err := km.DeleteAccounts(ctx, rawPublicKeys); err != nil {
			return errors.Wrap(err, "could not delete accounts")
		}
	case v2keymanager.Derived:
		return errors.New("cannot delete accounts for a derived keymanager")
	default:
		return fmt.Errorf("keymanager kind %s not supported", wallet.KeymanagerKind())
	}
	log.WithField("publicKeys", allAccountStr).Info("Accounts deleted")
	return nil
}
