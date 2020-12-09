package kv

import (
	"context"

	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SlashablePublicKeys returns keys that were marked as slashable during EIP-3076 slashing
// protection imports, ensuring that we can prevent these keys from having duties at runtime.
func (s *Store) SlashablePublicKeys(ctx context.Context) ([][48]byte, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.SlashablePublicKeys")
	defer span.End()
	var err error
	publicKeys := make([][48]byte, 0)
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(slashablePublicKeysBucket)
		return bucket.ForEach(func(key []byte, _ []byte) error {
			if key != nil {
				pubKeyBytes := [48]byte{}
				copy(pubKeyBytes[:], key)
				publicKeys = append(publicKeys, pubKeyBytes)
			}
			return nil
		})
	})
	return publicKeys, err
}

// SaveSlashablePublicKeys stores a list of slashable public keys that
// were determined during EIP-3076 slashing protection imports.
func (s *Store) SaveSlashablePublicKeys(ctx context.Context, publicKeys [][48]byte) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveSlashablePublicKeys")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(slashablePublicKeysBucket)
		for _, pubKey := range publicKeys {
			// We write the public key to disk in the bucket. The value written for the key does not
			// matter as we'll only be looking at the keys in the bucket when fetching from disk.
			if err := bkt.Put(pubKey[:], []byte{1}); err != nil {
				return err
			}
		}
		return nil
	})
}
