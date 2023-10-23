package accounts

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/v4/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v4/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v4/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/v4/validator/keymanager/local"
)

// WalletCreate creates wallet specified by configuration options.
func (acm *CLIManager) WalletCreate(ctx context.Context) (*wallet.Wallet, error) {
	w := wallet.New(&wallet.Config{
		WalletDir:      acm.walletDir,
		KeymanagerKind: acm.keymanagerKind,
		WalletPassword: acm.walletPassword,
	})
	var err error
	switch w.KeymanagerKind() {
	case keymanager.Local:
		if err = CreateLocalKeymanagerWallet(ctx, w); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet")
		}
		// TODO(#9883) - Remove this when we have a better way to handle this. should be safe to use for now.
		km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
		if err != nil {
			return nil, errors.Wrap(err, ErrCouldNotInitializeKeymanager)
		}
		localKm, ok := km.(*local.Keymanager)
		if !ok {
			return nil, errors.Wrap(err, ErrCouldNotInitializeKeymanager)
		}
		accountsKeystore, err := localKm.CreateAccountsKeystore(ctx, make([][]byte, 0), make([][]byte, 0))
		if err != nil {
			return nil, err
		}
		encodedAccounts, err := json.MarshalIndent(accountsKeystore, "", "\t")
		if err != nil {
			return nil, err
		}
		if err = w.WriteFileAtPath(ctx, local.AccountsPath, local.AccountsKeystoreFileName, encodedAccounts); err != nil {
			return nil, err
		}

		log.WithField("--wallet-dir", acm.walletDir).Info(
			"Successfully created wallet with ability to import keystores",
		)
	case keymanager.Derived:
		if err = createDerivedKeymanagerWallet(
			ctx,
			w,
			acm.mnemonic25thWord,
			acm.mnemonicLanguage,
			acm.skipMnemonicConfirm,
			acm.numAccounts,
		); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet")
		}
		log.WithField("--wallet-dir", acm.walletDir).Info(
			"Successfully created HD wallet from mnemonic and regenerated accounts",
		)
	case keymanager.Web3Signer:
		return nil, errors.New("web3signer keymanager does not require persistent wallets.")
	default:
		return nil, errors.Wrapf(err, errKeymanagerNotSupported, w.KeymanagerKind())
	}
	return w, nil
}

func CreateLocalKeymanagerWallet(_ context.Context, wallet *wallet.Wallet) error {
	if wallet == nil {
		return errors.New("nil wallet")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet to disk")
	}
	return nil
}

func createDerivedKeymanagerWallet(
	ctx context.Context,
	wallet *wallet.Wallet,
	mnemonicPassphrase string,
	mnemonicLanguage string,
	skipMnemonicConfirm bool,
	numAccounts int,
) error {
	if wallet == nil {
		return errors.New("nil wallet")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet to disk")
	}
	km, err := derived.NewKeymanager(ctx, &derived.SetupConfig{
		Wallet:           wallet,
		ListenForChanges: true,
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize HD keymanager")
	}
	mnemonic, err := derived.GenerateAndConfirmMnemonic(mnemonicLanguage, skipMnemonicConfirm)
	if err != nil {
		return errors.Wrap(err, "could not confirm mnemonic")
	}
	if err := km.RecoverAccountsFromMnemonic(ctx, mnemonic, mnemonicLanguage, mnemonicPassphrase, numAccounts); err != nil {
		return errors.Wrap(err, "could not recover accounts from mnemonic")
	}
	return nil
}
