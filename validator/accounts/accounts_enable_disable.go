package accounts

import (
	"context"
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
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/imported"
	"github.com/urfave/cli/v2"
)

func DisableAccountsCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, wallet.ErrNoWalletFound
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	keymanager, err := w.InitializeKeymanager(cliCtx.Context, &iface.InitializeKeymanagerConfig{
		SkipMnemonicConfirm: false,
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	validatingPublicKeys, err := keymanager.FetchValidatingPublicKeys(cliCtx.Context)
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
			if strings.ToLower(resp) == "n" {
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
			if strings.ToLower(resp) == "n" {
				return nil
			}
		}
	}
	if err := DisableAccounts(cliCtx.Context, &AccountsConfig{
		Wallet:            w,
		Keymanager:        keymanager,
		DisablePublicKeys: rawPublicKeys,
	}); err != nil {
		return err
	}
	log.WithField("publicKeys", allAccountStr).Info("Accounts disabled")
	return nil
}

func EnableAccountsCli(cliCtx *cli.Context) error {
	w, err := wallet.OpenWalletOrElseCli(cliCtx, func(cliCtx *cli.Context) (*wallet.Wallet, error) {
		return nil, wallet.ErrNoWalletFound
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	ikeymanager, err := w.InitializeKeymanager(cliCtx.Context, &iface.InitializeKeymanagerConfig{
		SkipMnemonicConfirm: false,
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	switch w.KeymanagerKind() {
	case keymanager.Remote:
		return errors.New("cannot enable accounts for a remote keymanager")
	case keymanager.Imported:
		km, ok := ikeymanager.(*imported.Keymanager)
		if !ok {
			return errors.New("not a imported keymanager")
		}
		disabledPublicKeys := km.KeymanagerOpts().DisabledPublicKeys
		if len(disabledPublicKeys) == 0 {
			return errors.New("No accounts are disabled.")
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
		if !cliCtx.IsSet(flags.DisablePublicKeysFlag.Name) {
			if len(filteredPubKeys) == 1 {
				promptText := "Are you sure you want to enable 1 account? (%s) Y/N"
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
				if strings.ToLower(resp) == "n" {
					return nil
				}
			}
		}
		if err := EnableAccounts(cliCtx.Context, &AccountsConfig{
			Wallet:           w,
			Keymanager:       ikeymanager,
			EnablePublicKeys: rawPublicKeys,
		}); err != nil {
			return err
		}
		log.WithField("publicKeys", allAccountStr).Info("Accounts enabled")
		return nil
	case keymanager.Derived:
		return errors.New("cannot enable accounts for a derived keymanager")
	default:
		return fmt.Errorf("keymanager kind %s not supported", w.KeymanagerKind())
	}
}

// DisableAccount disable the accounts that the user requests to be disabled from the wallet
func DisableAccounts(ctx context.Context, cfg *AccountsConfig) error {
	switch cfg.Wallet.KeymanagerKind() {
	case keymanager.Remote:
		return errors.New("cannot disable accounts for a remote keymanager")
	case keymanager.Imported:
		km, ok := cfg.Keymanager.(*imported.Keymanager)
		if !ok {
			return errors.New("not a imported keymanager")
		}
		if len(cfg.DisablePublicKeys) == 1 {
			log.Info("Disabling account...")
		} else {
			log.Info("Disabling accounts...")
		}
		updatedOpts := km.KeymanagerOpts()
		// updatedDisabledPubKeys := make([][48]byte, 0)
		existingDisabledPubKeys := make(map[[48]byte]bool, len(updatedOpts.DisabledPublicKeys))
		for _, pk := range updatedOpts.DisabledPublicKeys {
			existingDisabledPubKeys[bytesutil.ToBytes48(pk)] = true
		}
		for _, pk := range cfg.DisablePublicKeys {
			if _, ok := existingDisabledPubKeys[bytesutil.ToBytes48(pk)]; !ok {
				updatedOpts.DisabledPublicKeys = append(updatedOpts.DisabledPublicKeys, pk)
			}
		}
		keymanagerConfig, err := imported.MarshalOptionsFile(ctx, updatedOpts)
		if err != nil {
			return errors.Wrap(err, "could not marshal keymanager config file")
		}
		if err := cfg.Wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
			return errors.Wrap(err, "could not write keymanager config to disk")
		}
	case keymanager.Derived:
		return errors.New("cannot disable accounts for a derived keymanager")
	default:
		return fmt.Errorf("keymanager kind %s not supported", cfg.Wallet.KeymanagerKind())
	}
	return nil
}

// EnableAccounts enable the accounts that the user requests to be enabled from the wallet
func EnableAccounts(ctx context.Context, cfg *AccountsConfig) error {
	switch cfg.Wallet.KeymanagerKind() {
	case keymanager.Remote:
		return errors.New("cannot enable accounts for a remote keymanager")
	case keymanager.Imported:
		km, ok := cfg.Keymanager.(*imported.Keymanager)
		if !ok {
			return errors.New("not a imported keymanager")
		}
		if len(cfg.EnablePublicKeys) == 1 {
			log.Info("Enabling account...")
		} else {
			log.Info("Enabling accounts...")
		}
		updatedOpts := km.KeymanagerOpts()
		updatedDisabledPubKeys := make([][]byte, 0)
		setEnablePubKeys := make(map[[48]byte]bool, len(cfg.EnablePublicKeys))
		for _, pk := range cfg.EnablePublicKeys {
			setEnablePubKeys[bytesutil.ToBytes48(pk)] = true
		}
		for _, pk := range updatedOpts.DisabledPublicKeys {
			if _, ok := setEnablePubKeys[bytesutil.ToBytes48(pk)]; !ok {
				updatedDisabledPubKeys = append(updatedDisabledPubKeys, pk)
			}
		}
		updatedOpts.DisabledPublicKeys = updatedDisabledPubKeys
		keymanagerConfig, err := imported.MarshalOptionsFile(ctx, updatedOpts)
		if err != nil {
			return errors.Wrap(err, "could not marshal keymanager config file")
		}
		if err := cfg.Wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
			return errors.Wrap(err, "could not write keymanager config to disk")
		}
	case keymanager.Derived:
		return errors.New("cannot enable accounts for a derived keymanager")
	default:
		return fmt.Errorf("keymanager kind %s not supported", cfg.Wallet.KeymanagerKind())
	}
	return nil
}
