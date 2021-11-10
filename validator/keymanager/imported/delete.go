package imported

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	"github.com/prysmaticlabs/prysm/validator/accounts/petnames"
	"github.com/sirupsen/logrus"
)

// DeleteKeystores takes in public keys and removes the accounts from the wallet.
// This includes their disk keystore and cached keystore, but maintains the slashing
// protection history in the database.
func (km *Keymanager) DeleteKeystores(
	ctx context.Context, publicKeys [][]byte,
) ([]*ethpbservice.DeletedKeystoreStatus, error) {
	statuses := make([]*ethpbservice.DeletedKeystoreStatus, len(publicKeys))
	for i, publicKey := range publicKeys {
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
			statuses[i] = &ethpbservice.DeletedKeystoreStatus{
				Status: ethpbservice.DeletedKeystoreStatus_NOT_FOUND,
			}
			continue
		}
		deletedPublicKey := km.accountsStore.PublicKeys[index]
		accountName := petnames.DeterministicName(deletedPublicKey, "-")
		km.accountsStore.PrivateKeys = append(km.accountsStore.PrivateKeys[:index], km.accountsStore.PrivateKeys[index+1:]...)
		km.accountsStore.PublicKeys = append(km.accountsStore.PublicKeys[:index], km.accountsStore.PublicKeys[index+1:]...)

		newStore, err := km.CreateAccountsKeystore(ctx, km.accountsStore.PrivateKeys, km.accountsStore.PublicKeys)
		if err != nil {
			return nil, errors.Wrap(err, "could not rewrite accounts keystore")
		}

		// Write the encoded keystore.
		encoded, err := json.MarshalIndent(newStore, "", "\t")
		if err != nil {
			return nil, err
		}
		if err := km.wallet.WriteFileAtPath(ctx, AccountsPath, AccountsKeystoreFileName, encoded); err != nil {
			return nil, errors.Wrap(err, "could not write keystore file for accounts")
		}

		log.WithFields(logrus.Fields{
			"name":      accountName,
			"publicKey": fmt.Sprintf("%#x", bytesutil.Trunc(deletedPublicKey)),
		}).Info("Successfully deleted validator account")
		err = km.initializeKeysCachesFromKeystore()
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize keys caches")
		}
		statuses[i] = &ethpbservice.DeletedKeystoreStatus{
			Status: ethpbservice.DeletedKeystoreStatus_DELETED,
		}
	}
	return statuses, nil
}
