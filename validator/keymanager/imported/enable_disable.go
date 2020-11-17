package imported

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// DisableAccounts disables public keys from the user's wallet.
func (dr *Keymanager) DisableAccounts(ctx context.Context, pubKeys [][]byte) error {
	if pubKeys == nil || len(pubKeys) < 1 {
		return errors.New("no public keys specified to disable")
	}
	existingDisabledPubKeys := make(map[[48]byte]bool)
	for _, pk := range dr.disabledPublicKeys {
		existingDisabledPubKeys[bytesutil.ToBytes48(pk)] = true
	}
	for _, pk := range pubKeys {
		if _, ok := existingDisabledPubKeys[bytesutil.ToBytes48(pk)]; !ok {
			existingDisabledPubKeys[bytesutil.ToBytes48(pk)] = true
		}
	}
	newlyDisabledPubKeys := make([][]byte, 0)
	for pk := range existingDisabledPubKeys {
		newlyDisabledPubKeys = append(newlyDisabledPubKeys, pk[:])
	}
	dr.disabledPublicKeys = newlyDisabledPubKeys
	return nil
}

// EnableAccounts enables public keys from a user's wallet if they are disabled.
func (dr *Keymanager) EnableAccounts(ctx context.Context, pubKeys [][]byte) error {
	if pubKeys == nil || len(pubKeys) < 1 {
		return errors.New("no public keys specified to enable")
	}
	existingDisabledPubKeys := make(map[[48]byte]bool)
	for _, pk := range dr.disabledPublicKeys {
		existingDisabledPubKeys[bytesutil.ToBytes48(pk)] = true
	}
	for _, pk := range pubKeys {
		keyBytes := bytesutil.ToBytes48(pk)
		if _, ok := existingDisabledPubKeys[keyBytes]; ok {
			delete(existingDisabledPubKeys, keyBytes)
		}
	}
	newlyDisabledPubKeys := make([][]byte, 0)
	for pk := range existingDisabledPubKeys {
		newlyDisabledPubKeys = append(newlyDisabledPubKeys, pk[:])
	}
	dr.disabledPublicKeys = newlyDisabledPubKeys
	// TODO: Write keystores.
	return nil
}
