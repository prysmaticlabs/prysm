package imported

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	"github.com/sirupsen/logrus"
)

// DeleteKeystores takes in public keys and removes the accounts from the wallet.
// This includes their disk keystore and cached keystore, but maintains the slashing
// protection history in the database.
func (km *Keymanager) DeleteKeystores(
	ctx context.Context, publicKeys [][]byte,
) ([]*ethpbservice.DeletedKeystoreStatus, error) {
	// Check for duplicate keys and filter them out.
	trackedPublicKeys := make(map[[48]byte]bool)
	statuses := make([]*ethpbservice.DeletedKeystoreStatus, 0, len(publicKeys))
	var store *AccountsKeystoreRepresentation
	var err error
	deletedKeys := make([][]byte, 0, len(publicKeys))
	for _, publicKey := range publicKeys {
		// Check if the key in the request is a duplicate.
		if _, ok := trackedPublicKeys[bytesutil.ToBytes48(publicKey)]; ok {
			statuses = append(statuses, &ethpbservice.DeletedKeystoreStatus{
				Status: ethpbservice.DeletedKeystoreStatus_NOT_ACTIVE,
			})
			continue
		}
		var index int
		var found bool
		for j, pubKey := range km.accountsStore.PublicKeys {
			if bytes.Equal(pubKey, publicKey) {
				index = j
				found = true
				break
			}
		}
		if !found {
			statuses = append(statuses, &ethpbservice.DeletedKeystoreStatus{
				Status: ethpbservice.DeletedKeystoreStatus_NOT_FOUND,
			})
			continue
		}
		deletedPublicKey := km.accountsStore.PublicKeys[index]
		deletedKeys = append(deletedKeys, deletedPublicKey)
		km.accountsStore.PrivateKeys = append(km.accountsStore.PrivateKeys[:index], km.accountsStore.PrivateKeys[index+1:]...)
		km.accountsStore.PublicKeys = append(km.accountsStore.PublicKeys[:index], km.accountsStore.PublicKeys[index+1:]...)
		store, err = km.CreateAccountsKeystore(ctx, km.accountsStore.PrivateKeys, km.accountsStore.PublicKeys)
		if err != nil {
			return nil, errors.Wrap(err, "could not rewrite accounts keystore")
		}
		statuses = append(statuses, &ethpbservice.DeletedKeystoreStatus{
			Status: ethpbservice.DeletedKeystoreStatus_DELETED,
		})
		trackedPublicKeys[bytesutil.ToBytes48(publicKey)] = true
	}
	if len(deletedKeys) == 0 {
		return statuses, nil
	}
	var deletedKeysStr string
	for i, k := range deletedKeys {
		if i == 0 {
			deletedKeysStr += fmt.Sprintf("%#x", bytesutil.Trunc(k))
		} else if i == len(deletedKeys)-1 {
			deletedKeysStr += fmt.Sprintf("%#x", bytesutil.Trunc(k))
		} else {
			deletedKeysStr += fmt.Sprintf(",%#x", bytesutil.Trunc(k))
		}
	}

	log.WithFields(logrus.Fields{
		"publicKeys": deletedKeysStr,
	}).Info("Successfully deleted validator key(s)")

	// Write the encoded keystore.
	encoded, err := json.MarshalIndent(store, "", "\t")
	if err != nil {
		return nil, err
	}
	if err := km.wallet.WriteFileAtPath(ctx, AccountsPath, AccountsKeystoreFileName, encoded); err != nil {
		return nil, errors.Wrap(err, "could not write keystore file for accounts")
	}
	err = km.initializeKeysCachesFromKeystore()
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize key caches")
	}
	log.WithFields(logrus.Fields{
		"publicKeys": deletedKeysStr,
	}).Info("Successfully deleted validator key(s)")
	return statuses, nil
}
