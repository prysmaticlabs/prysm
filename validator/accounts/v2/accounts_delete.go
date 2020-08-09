package v2

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/urfave/cli/v2"
)

// DeleteAccount deletes the accounts that the user requests to be deleted from the wallet.
func DeleteAccount(cliCtx *cli.Context) error {
	ctx := context.Background()
	wallet, err := OpenWallet(cliCtx)
	if err != nil {
		return err
	}
	skipMnemonicConfirm := cliCtx.Bool(flags.SkipMnemonicConfirmFlag.Name)
	keymanager, err := wallet.InitializeKeymanager(ctx, skipMnemonicConfirm)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	allAccounts, err := keymanager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return err
	}
	var allAccountStrs []string
	for _, account := range allAccounts {
		allAccountStrs = append(allAccountStrs, fmt.Sprintf("%#x", bytesutil.FromBytes48(account)))
	}
	accounts, err := selectAccounts(cliCtx, selectAccountsDeletePromptText, allAccountStrs)
	if err != nil {
		return err
	}
	fmt.Println(strings.Join(accounts, ", "))
	if len(accounts) == 1 {
		log.Info("Deleting account...")
	} else {
		log.Info("Deleting accounts...")
	}
	switch wallet.KeymanagerKind() {
	case v2keymanager.Remote:
		return errors.New("cannot create a new account for a remote keymanager")
	case v2keymanager.Direct:
		km, ok := keymanager.(*direct.Keymanager)
		if !ok {
			return errors.New("not a direct keymanager")
		}
		// Delete the requested account's keystore.
		for _, account := range accounts {
			pubKeyBytes, err := hex.DecodeString(account[2:])
			if err != nil {
				return errors.Wrapf(err, "could not decode public key %s", account)
			}
			if err := km.DeleteAccount(ctx, pubKeyBytes); err != nil {
				return errors.Wrap(err, "could not create account in wallet")
			}
		}
	case v2keymanager.Derived:
		return errors.New("cannot create a new account for a derived keymanager")
	default:
		return fmt.Errorf("keymanager kind %s not supported", wallet.KeymanagerKind())
	}
	formattedPubKeys := make([]string, len(accounts))
	for i, account := range accounts {
		formattedPubKeys[i] = account[:8]
	}
	allAccountStr := strings.Join(formattedPubKeys, ", ")
	log.WithField("publicKeys", allAccountStr).Info("Accounts deleted")
	return nil
}
