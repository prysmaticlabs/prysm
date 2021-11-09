package imported

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/petnames"
	"github.com/sirupsen/logrus"
)

// DeleteAccounts takes in public keys and removes the accounts entirely. This includes their disk keystore and cached keystore.
func (km *Keymanager) DeleteAccounts(ctx context.Context, publicKeys [][]byte) error {
	for _, publicKey := range publicKeys {
		var index int
		var found bool
		for i, pubKey := range km.accountsStore.PublicKeys {
			if bytes.Equal(pubKey, publicKey) {
				index = i
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("could not find public key %#x", publicKey)
		}
		deletedPublicKey := km.accountsStore.PublicKeys[index]
		accountName := petnames.DeterministicName(deletedPublicKey, "-")
		km.accountsStore.PrivateKeys = append(km.accountsStore.PrivateKeys[:index], km.accountsStore.PrivateKeys[index+1:]...)
		km.accountsStore.PublicKeys = append(km.accountsStore.PublicKeys[:index], km.accountsStore.PublicKeys[index+1:]...)

		newStore, err := km.CreateAccountsKeystore(ctx, km.accountsStore.PrivateKeys, km.accountsStore.PublicKeys)
		if err != nil {
			return errors.Wrap(err, "could not rewrite accounts keystore")
		}

		// Write the encoded keystore.
		encoded, err := json.MarshalIndent(newStore, "", "\t")
		if err != nil {
			return err
		}
		if err := km.wallet.WriteFileAtPath(ctx, AccountsPath, AccountsKeystoreFileName, encoded); err != nil {
			return errors.Wrap(err, "could not write keystore file for accounts")
		}

		log.WithFields(logrus.Fields{
			"name":      accountName,
			"publicKey": fmt.Sprintf("%#x", bytesutil.Trunc(deletedPublicKey)),
		}).Info("Successfully deleted validator account")
		err = km.initializeKeysCachesFromKeystore()
		if err != nil {
			return errors.Wrap(err, "failed to initialize keys caches")
		}
	}
	return nil
}
