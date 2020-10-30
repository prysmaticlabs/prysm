package imported

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/asyncutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var (
	debounceFileChangesInterval = time.Second
)

// Listen for changes to the all-accounts.keystore.json file in our wallet
// to load in new keys we observe into our keymanager. This uses the fsnotify
// library to listen for file-system changes and debounces these events to
// ensure we can handle thousands of events fired in a short time-span.
func (dr *Keymanager) listenForAccountChanges(ctx context.Context) {
	accountsFilePath := filepath.Join(dr.wallet.AccountsDir(), AccountsPath, accountsKeystoreFileName)
	if !fileutil.FileExists(accountsFilePath) {
		return
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Error("Could not initialize file watcher")
		return
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.WithError(err).Error("Could not close file watcher")
		}
	}()
	if err := watcher.Add(accountsFilePath); err != nil {
		log.WithError(err).Errorf("Could not add file %s to file watcher", accountsFilePath)
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	fileChangesChan := make(chan interface{}, 100)
	defer close(fileChangesChan)

	// We debounce events sent over the file changes channel by an interval
	// to ensure we are not overwhelmed by a ton of events fired over the channel in
	// a short span of time.
	go asyncutil.Debounce(ctx, debounceFileChangesInterval, fileChangesChan, func(event interface{}) {
		ev, ok := event.(fsnotify.Event)
		if !ok {
			log.Errorf("Type %T is not a valid file system event", event)
			return
		}
		fileBytes, err := ioutil.ReadFile(ev.Name)
		if err != nil {
			log.WithError(err).Errorf("Could not read file at path: %s", ev.Name)
			return
		}
		accountsKeystore := &keymanager.Keystore{}
		if err := json.Unmarshal(fileBytes, accountsKeystore); err != nil {
			log.WithError(
				err,
			).Errorf("Could not read valid, EIP-2335 keystore json file at path: %s", ev.Name)
			return
		}
		if err := dr.reloadAccountsFromKeystore(accountsKeystore); err != nil {
			log.WithError(
				err,
			).Error("Could not replace the accounts store from keystore file")
		}
	})
	for {
		select {
		case event := <-watcher.Events:
			// If a file was modified, we attempt to read that file
			// and parse it into our accounts store.
			if event.Op&fsnotify.Write == fsnotify.Write {
				fileChangesChan <- event
			}
		case err := <-watcher.Errors:
			log.WithError(err).Errorf("Could not watch for file changes for: %s", accountsFilePath)
		case <-ctx.Done():
			return
		}
	}
}

// Replaces the accounts store struct in the imported keymanager with
// the contents of a keystore file by decrypting it with the accounts password.
func (dr *Keymanager) reloadAccountsFromKeystore(keystore *keymanager.Keystore) error {
	decryptor := keystorev4.New()
	encodedAccounts, err := decryptor.Decrypt(keystore.Crypto, dr.wallet.Password())
	if err != nil {
		return errors.Wrapf(err, "could not decrypt keystore file with public key %s", keystore.Pubkey)
	}
	newAccountsStore := &AccountStore{}
	if err := json.Unmarshal(encodedAccounts, newAccountsStore); err != nil {
		return err
	}
	dr.accountsStore = newAccountsStore
	pubKeys := make([][48]byte, len(dr.accountsStore.PublicKeys))
	for i := 0; i < len(dr.accountsStore.PrivateKeys); i++ {
		privKey, err := bls.SecretKeyFromBytes(dr.accountsStore.PrivateKeys[i])
		if err != nil {
			return errors.Wrap(err, "could not initialize private key")
		}
		pubKeyBytes := privKey.PublicKey().Marshal()
		pubKeys[i] = bytesutil.ToBytes48(pubKeyBytes)
	}
	if err := dr.initializeKeysCachesFromKeystore(); err != nil {
		return err
	}
	log.Info("Reloaded validator keys into keymanager")
	dr.accountsChangedFeed.Send(pubKeys)
	return nil
}
