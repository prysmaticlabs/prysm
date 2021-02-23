package accounts

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/validator/accounts/prompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/urfave/cli/v2"
)

// DisableAccountsCli disables via CLI the accounts that the user requests to be disabled from the wallet
func DisableAccountsCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, wallet.ErrNoWalletFound
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	km, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	if err != nil {
		return errors.Wrap(err, ErrCouldNotInitializeKeymanager)
	}
	validatingPublicKeys, err := km.FetchValidatingPublicKeys(cliCtx.Context)
	if err != nil {
		return err
	}
	if len(validatingPublicKeys) == 0 {
		return errors.New("wallet is empty, no accounts to disable")
	}
	// Allow the user to interactively select the accounts to disable or optionally
	// provide them via cli flags as a string of comma-separated, hex strings.
	filteredPubKeys, err := filterPublicKeysFromUserInput(
		cliCtx,
		flags.DisablePublicKeysFlag,
		validatingPublicKeys,
		prompt.SelectAccountsDisablePromptText,
	)
	if err != nil {
		return errors.Wrap(err, "could not filter public keys for deactivation")
	}
	rawPublicKeys := make([][]byte, len(filteredPubKeys))
	formattedPubKeys := make([]string, len(filteredPubKeys))
	for i, pk := range filteredPubKeys {
		pubKeyBytes := pk.Marshal()
		rawPublicKeys[i] = pubKeyBytes
		formattedPubKeys[i] = fmt.Sprintf("%#x", bytesutil.Trunc(pubKeyBytes))
	}
	allAccountStr := strings.Join(formattedPubKeys, ", ")
	if !cliCtx.IsSet(flags.DisablePublicKeysFlag.Name) {
		if len(filteredPubKeys) == 1 {
			promptText := "Are you sure you want to disable 1 account? (%s) Y/N"
			resp, err := promptutil.ValidatePrompt(
				os.Stdin, fmt.Sprintf(promptText, au.BrightGreen(formattedPubKeys[0])), promptutil.ValidateYesOrNo,
			)
			if err != nil {
				return err
			}
			if strings.EqualFold(resp, "n") {
				return nil
			}
		} else {
			promptText := "Are you sure you want to disable %d accounts? (%s) Y/N"
			if len(filteredPubKeys) == len(validatingPublicKeys) {
				promptText = fmt.Sprintf("Are you sure you want to disable all accounts? Y/N (%s)", au.BrightGreen(allAccountStr))
			} else {
				promptText = fmt.Sprintf(promptText, len(filteredPubKeys), au.BrightGreen(allAccountStr))
			}
			resp, err := promptutil.ValidatePrompt(os.Stdin, promptText, promptutil.ValidateYesOrNo)
			if err != nil {
				return err
			}
			if strings.EqualFold(resp, "n") {
				return nil
			}
		}
	}
	importedKM, ok := km.(*imported.Keymanager)
	if !ok {
		return errors.New("can only disable accounts for imported wallets")
	}
	if err := importedKM.DisableAccounts(cliCtx.Context, rawPublicKeys); err != nil {
		return err
	}
	log.WithField("publicKeys", allAccountStr).Info("Accounts disabled")
	return nil
}

// EnableAccountsCli enables via CLI the accounts that the user requests to be enabled from the wallet
func EnableAccountsCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, wallet.ErrNoWalletFound
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	km, err := w.InitializeKeymanager(cliCtx.Context, iface.InitKeymanagerConfig{ListenForChanges: false})
	if err != nil {
		return errors.Wrap(err, ErrCouldNotInitializeKeymanager)
	}
	importedKM, ok := km.(*imported.Keymanager)
	if !ok {
		return errors.New("can only enable/disable accounts for imported wallets")
	}
	disabledPublicKeys := importedKM.DisabledPublicKeys()
	if len(disabledPublicKeys) == 0 {
		return errors.New("no accounts are disabled")
	}
	disabledPublicKeys48 := make([][48]byte, len(disabledPublicKeys))
	for i := range disabledPublicKeys {
		disabledPublicKeys48[i] = bytesutil.ToBytes48(disabledPublicKeys[i])
	}

	// Allow the user to interactively select the accounts to enable or optionally
	// provide them via cli flags as a string of comma-separated, hex strings.
	filteredPubKeys, err := filterPublicKeysFromUserInput(
		cliCtx,
		flags.EnablePublicKeysFlag,
		disabledPublicKeys48,
		prompt.SelectAccountsEnablePromptText,
	)
	if err != nil {
		return errors.Wrap(err, "could not filter public keys for activation")
	}
	rawPublicKeys := make([][]byte, len(filteredPubKeys))
	formattedPubKeys := make([]string, len(filteredPubKeys))
	for i, pk := range filteredPubKeys {
		pubKeyBytes := pk.Marshal()
		rawPublicKeys[i] = pubKeyBytes
		formattedPubKeys[i] = fmt.Sprintf("%#x", bytesutil.Trunc(pubKeyBytes))
	}
	allAccountStr := strings.Join(formattedPubKeys, ", ")
	if !cliCtx.IsSet(flags.EnablePublicKeysFlag.Name) {
		if len(filteredPubKeys) == 1 {
			promptText := "Are you sure you want to enable 1 account? (%s) Y/N"
			resp, err := promptutil.ValidatePrompt(
				os.Stdin, fmt.Sprintf(promptText, au.BrightGreen(formattedPubKeys[0])), promptutil.ValidateYesOrNo,
			)
			if err != nil {
				return err
			}
			if strings.EqualFold(resp, "n") {
				return nil
			}
		} else {
			promptText := "Are you sure you want to enable %d accounts? (%s) Y/N"
			if len(filteredPubKeys) == len(disabledPublicKeys48) {
				promptText = fmt.Sprintf("Are you sure you want to enable all accounts? Y/N (%s)", au.BrightGreen(allAccountStr))
			} else {
				promptText = fmt.Sprintf(promptText, len(filteredPubKeys), au.BrightGreen(allAccountStr))
			}
			resp, err := promptutil.ValidatePrompt(os.Stdin, promptText, promptutil.ValidateYesOrNo)
			if err != nil {
				return err
			}
			if strings.EqualFold(resp, "n") {
				return nil
			}
		}
	}
	if err := importedKM.EnableAccounts(cliCtx.Context, rawPublicKeys); err != nil {
		return err
	}
	log.WithField("publicKeys", allAccountStr).Info("Accounts enabled")
	return nil
}
