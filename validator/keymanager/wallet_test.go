package keymanager_test

import (
	"fmt"
	types "github.com/wealdtech/go-eth2-wallet-types/v2"
	"io/ioutil"
	"os"

	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/validator/keymanager"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
	nd "github.com/wealdtech/go-eth2-wallet-nd/v2"
	filesystem "github.com/wealdtech/go-eth2-wallet-store-filesystem"
)

func SetupWallet(t *testing.T) (string, types.Wallet) {
	path, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	store := filesystem.New(filesystem.WithLocation(path))
	encryptor := keystorev4.New()

	// Create wallets with keys.
	w1, err := nd.CreateWallet("Wallet 1", store, encryptor)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	err = w1.Unlock(nil)
	if err != nil {
		t.Fatalf("Failed to unlock wallet: %v", err)
	}
	_, err = addAccount(w1, "Account 1", "foo")
	if err != nil {
		t.Fatalf("Failed to create account 1: %v", err)
	}
	_, err = addAccount(w1, "Account 2", "bar")
	if err != nil {
		t.Fatalf("Failed to create account 2: %v", err)
	}

	return path,w1
}

func addAccount(wallet types.Wallet, name string, password string) (types.Account,error) {
	return wallet.CreateAccount(name, []byte(password))
}

func removeAccount(wallet types.Wallet, account types.Account, path string) error {
	finalpath := path + "/" + wallet.ID().String() + "/" + account.ID().String()
	return os.Remove(finalpath)
}

func wallet(t *testing.T, opts string) keymanager.KeyManager {
	km, _, err := keymanager.NewWallet(opts)
	if err != nil {
		t.Fatal(err)
	}
	return km
}

func TestMultiplePassphrases(t *testing.T) {
	path,_ := SetupWallet(t)
	//defer os.RemoveAll(path)
	tests := []struct {
		name     string
		wallet   keymanager.KeyManager
		accounts int
	}{
		{
			name:     "Neither",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["neither"]}`, path)),
			accounts: 0,
		},
		{
			name:     "Foo",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["foo"]}`, path)),
			accounts: 1,
		},
		{
			name:     "Bar",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["bar"]}`, path)),
			accounts: 1,
		},
		{
			name:     "Both",
			wallet:   wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["foo","bar"]}`, path)),
			accounts: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keys, err := test.wallet.FetchValidatingKeys()
			if err != nil {
				t.Error(err)
			}
			if len(keys) != test.accounts {
				t.Errorf("Found %d keys; expected %d", len(keys), test.accounts)
			}
		})
	}

	err := os.RemoveAll(path)
	if err != nil {
		t.Error(err)
	}
}

func TestRemovingAccountsDynamically(t *testing.T) {
	t.Run("Remove account dynamically", func(t *testing.T) {
		path, ndWallet := SetupWallet(t)
		wallet := wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["foo","bar"]}`, path))

		accountToDelete := <-ndWallet.Accounts()
		err := removeAccount(ndWallet, accountToDelete,path)
		if err != nil {
			t.Error(err)
		}

		time.Sleep(100 * time.Millisecond) // necessary to ensure wallet has enough time to fetch new accounts
		keys, err := wallet.FetchValidatingKeys()
		if err != nil {
			t.Error(err)
		}

		if len(keys) != 1 {
			t.Errorf("Found %d keys; expected %d", len(keys), 1)
		}

		err = os.RemoveAll(path)
		if err != nil {
			t.Error(err)
		}
	})
}

func TestAddingAccountsDynamically(t *testing.T) {
	tests := []struct {
		name                  string
		existingAccounts      int
		newAccounts           int
		passphrases           []string
		expectedFinalAccounts int
	}{
		{
			name:                  "add 1 new account, same passphrase",
			existingAccounts:      2,
			newAccounts:           1,
			passphrases:           []string{"foo"},
			expectedFinalAccounts: 3,
		},
		{
			name:                  "add 1 new account, unknown passphrase",
			existingAccounts:      2,
			newAccounts:           1,
			passphrases:           []string{"unknown"},
			expectedFinalAccounts: 2,
		},
		{
			name:                  "bulk adding",
			existingAccounts:      2,
			newAccounts:           5,
			passphrases:           []string{"foo","foo","foo","foo","foo"},
			expectedFinalAccounts: 7,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, ndWallet := SetupWallet(t)
			wallet := wallet(t, fmt.Sprintf(`{"location":%q,"accounts":["Wallet 1"],"passphrases":["foo","bar"]}`, path))

			// add new existingAccounts
			for i := 0 ; i < test.newAccounts; i++ {
				newAccountID := i+test.existingAccounts+1
				_,error := addAccount(ndWallet, fmt.Sprintf("Account %d",newAccountID), test.passphrases[i])
				if error != nil {
					t.Error(error)
				}
			}
			time.Sleep(100 * time.Millisecond) // necessary to ensure wallet has enough time to fetch new accounts
			keys, err := wallet.FetchValidatingKeys()
			if err != nil {
				t.Error(err)
			}

			if len(keys) != test.expectedFinalAccounts {
				t.Errorf("Found %d keys; expected %d", len(keys), test.expectedFinalAccounts)
			}

			err = os.RemoveAll(path)
			if err != nil {
				t.Error(err)
			}
		})
	}
}