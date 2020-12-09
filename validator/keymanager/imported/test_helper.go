package imported

import (
	"context"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/event"
	mock "github.com/prysmaticlabs/prysm/validator/accounts/testing"
)

// PrepareKeymanagerForKeystoreReload takes a keymanager as input
// and configures it so that it can be used in SimulateReloadingAccountsFromKeystore.
func PrepareKeymanagerForKeystoreReload(km *Keymanager) {
	password := "Passw03rdz293**%#2"
	wallet := &mock.Wallet{
		Files:            make(map[string]map[string][]byte),
		AccountPasswords: make(map[string]string),
		WalletPassword:   password,
	}
	km.wallet = wallet
	km.accountsChangedFeed = new(event.Feed)
}

// SimulateReloadingAccountsFromKeystore simulates the process of reloading accounts from a keystore.
func SimulateReloadingAccountsFromKeystore(km *Keymanager, privKeys []bls.SecretKey) error {
	numAccounts := len(privKeys)
	privKeyBytes := make([][]byte, numAccounts)
	pubKeyBytes := make([][]byte, numAccounts)
	for i := 0; i < numAccounts; i++ {
		privKeyBytes[i] = privKeys[i].Marshal()
		pubKeyBytes[i] = privKeys[i].PublicKey().Marshal()
	}

	accountsStore, err := km.createAccountsKeystore(context.Background(), privKeyBytes, pubKeyBytes)
	if err != nil {
		return err
	}
	if err = km.reloadAccountsFromKeystore(accountsStore); err != nil {
		return err
	}

	return nil
}
