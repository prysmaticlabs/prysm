package imported

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// DisableAccounts disables public keys from the user's wallet.
func (km *Keymanager) DisableAccounts(ctx context.Context, pubKeys [][]byte) error {
	if pubKeys == nil || len(pubKeys) < 1 {
		return errors.New("no public keys specified to disable")
	}
	lock.Lock()
	defer lock.Unlock()
	for _, pk := range pubKeys {
		if _, ok := km.disabledPublicKeys[bytesutil.ToBytes48(pk)]; !ok {
			km.disabledPublicKeys[bytesutil.ToBytes48(pk)] = true
		}
	}
	return km.rewriteDisabledKeysToDisk(ctx)
}

// EnableAccounts enables public keys from a user's wallet if they are disabled.
func (km *Keymanager) EnableAccounts(ctx context.Context, pubKeys [][]byte) error {
	if pubKeys == nil || len(pubKeys) < 1 {
		return errors.New("no public keys specified to enable")
	}
	lock.Lock()
	defer lock.Unlock()
	for _, pk := range pubKeys {
		delete(km.disabledPublicKeys, bytesutil.ToBytes48(pk))
	}
	return km.rewriteDisabledKeysToDisk(ctx)
}

func (km *Keymanager) rewriteDisabledKeysToDisk(ctx context.Context) error {
	encoded, err := km.wallet.ReadFileAtPath(ctx, AccountsPath, AccountsKeystoreFileName)
	if err != nil {
		return errors.Wrap(err, "could not read keystore file for accounts")
	}
	keystore := &AccountsKeystoreRepresentation{}
	if err := json.Unmarshal(encoded, keystore); err != nil {
		return err
	}
	disabledKeysStrings := make([]string, 0)
	for pk := range km.disabledPublicKeys {
		disabledKeysStrings = append(disabledKeysStrings, fmt.Sprintf("%x", pk))
	}
	keystore.DisabledPublicKeys = disabledKeysStrings
	encoded, err = json.MarshalIndent(keystore, "", "\t")
	if err != nil {
		return err
	}
	if err := km.wallet.WriteFileAtPath(ctx, AccountsPath, AccountsKeystoreFileName, encoded); err != nil {
		return errors.Wrap(err, "could not write keystore file for accounts")
	}
	return nil
}
