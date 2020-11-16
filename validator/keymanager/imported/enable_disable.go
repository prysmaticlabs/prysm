package imported

import (
	"context"
	"errors"
)

// DisableAccounts disables public keys from the user's wallet.
func (dr *Keymanager) DisableAccounts(ctx context.Context, pubKeys [][]byte) error {
	if pubKeys == nil || len(pubKeys) < 1 {
		return errors.New("no public keys specified to disable")
	}
	updatedOpts := km.KeymanagerOpts()
	// updatedDisabledPubKeys := make([][48]byte, 0)
	existingDisabledPubKeys := make(map[[48]byte]bool, len(updatedOpts.DisabledPublicKeys))
	for _, pk := range updatedOpts.DisabledPublicKeys {
		existingDisabledPubKeys[bytesutil.ToBytes48(pk)] = true
	}
	for _, pk := range cfg.DisablePublicKeys {
		if _, ok := existingDisabledPubKeys[bytesutil.ToBytes48(pk)]; !ok {
			updatedOpts.DisabledPublicKeys = append(updatedOpts.DisabledPublicKeys, pk)
		}
	}
	keymanagerConfig, err := imported.MarshalOptionsFile(ctx, updatedOpts)
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := cfg.Wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
}

// EnableAccounts enables public keys from a user's wallet if they are disabled.
func (dr *Keymanager) EnableAccounts(ctx context.Context, pubKeys [][]byte) error {
	if len(cfg.EnablePublicKeys) == 1 {
		log.Info("Enabling account...")
	} else {
		log.Info("Enabling accounts...")
	}
	updatedOpts := km.KeymanagerOpts()
	updatedDisabledPubKeys := make([][]byte, 0)
	setEnablePubKeys := make(map[[48]byte]bool, len(cfg.EnablePublicKeys))
	for _, pk := range cfg.EnablePublicKeys {
		setEnablePubKeys[bytesutil.ToBytes48(pk)] = true
	}
	for _, pk := range updatedOpts.DisabledPublicKeys {
		if _, ok := setEnablePubKeys[bytesutil.ToBytes48(pk)]; !ok {
			updatedDisabledPubKeys = append(updatedDisabledPubKeys, pk)
		}
	}
	updatedOpts.DisabledPublicKeys = updatedDisabledPubKeys
	keymanagerConfig, err := imported.MarshalOptionsFile(ctx, updatedOpts)
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := cfg.Wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
}
