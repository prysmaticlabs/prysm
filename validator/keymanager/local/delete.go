package local

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbservice "github.com/prysmaticlabs/prysm/v4/proto/eth/service"
	"github.com/sirupsen/logrus"
)

// DeleteKeystores takes in public keys and removes the accounts from the wallet.
// This includes their disk keystore and cached keystore, but maintains the slashing
// protection history in the database.
// 1) Copy the in memory keystore
// 2) Delete the keys from copied in memory keystore
// 3) Save the copy to disk
// 4) Reinitialize account store from disk
// 5) Verify keys are indeed deleted
// 6) Return API response
func (km *Keymanager) DeleteKeystores(
	ctx context.Context, publicKeys [][]byte,
) ([]*ethpbservice.DeletedKeystoreStatus, error) {
	// Check for duplicate keys and filter them out.
	trackedPublicKeys := make(map[[fieldparams.BLSPubkeyLength]byte]bool)
	statuses := make([]*ethpbservice.DeletedKeystoreStatus, 0, len(publicKeys))
	var store *AccountsKeystoreRepresentation
	var err error
	deletedKeys := make([][]byte, 0, len(publicKeys))
	// 1) Copy the in memory keystore
	storeCopy := km.accountsStore
	storeCopy.PrivateKeys = make([][]byte, len(km.accountsStore.PrivateKeys))
	copy(storeCopy.PrivateKeys, km.accountsStore.PrivateKeys)
	storeCopy.PublicKeys = make([][]byte, len(km.accountsStore.PublicKeys))
	copy(storeCopy.PublicKeys, km.accountsStore.PublicKeys)
	//
	for _, publicKey := range publicKeys {
		// Check if the key in the request is a duplicate or not found
		if _, ok := trackedPublicKeys[bytesutil.ToBytes48(publicKey)]; ok {
			statuses = append(statuses, &ethpbservice.DeletedKeystoreStatus{
				Status: ethpbservice.DeletedKeystoreStatus_NOT_ACTIVE,
			})
			continue
		}
		var index int
		var found bool
		for j, pubKey := range storeCopy.PublicKeys {
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
		//
		// 2) Delete the keys from copied in memory keystore
		deletedPublicKey := storeCopy.PublicKeys[index]
		deletedKeys = append(deletedKeys, deletedPublicKey)

		storeCopy.PrivateKeys = append(storeCopy.PrivateKeys[:index], storeCopy.PrivateKeys[index+1:]...)
		storeCopy.PublicKeys = append(storeCopy.PublicKeys[:index], storeCopy.PublicKeys[index+1:]...)
		store, err = km.CreateAccountsKeystore(ctx, storeCopy.PrivateKeys, storeCopy.PublicKeys)
		if err != nil {
			return nil, errors.Wrap(err, "could not rewrite accounts keystore")
		}
		//
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

	// 3) Save the copy to disk
	// Write the encoded keystore.
	encoded, err := json.MarshalIndent(store, "", "\t")
	if err != nil {
		return nil, err
	}
	if err := km.wallet.WriteFileAtPath(ctx, AccountsPath, AccountsKeystoreFileName, encoded); err != nil {
		return nil, errors.Wrap(err, "could not write keystore file for accounts")
	}
	//
	// 4) Reinitialize account store from disk

	//// ) Reload keystore Cache
	//err = km.initializeKeysCachesFromKeystore()
	//if err != nil {
	//	return nil, errors.Wrap(err, "failed to initialize key caches")
	//}
	////
	log.WithFields(logrus.Fields{
		"publicKeys": deletedKeysStr,
	}).Info("Successfully deleted validator key(s)")
	// 6) Return API response
	return statuses, nil
}
