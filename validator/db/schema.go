package db

import (
	"bytes"

	"github.com/boltdb/bolt"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// The Schema will define how to store and retrieve data from the db.
// Currently we store blocks by prefixing `block` to their hash and
// using that as the key to store blocks.
// `block` + hash -> block
//
// We store the state using the state lookup key, and
// also the genesis block using the genesis lookup key.
// The canonical head is stored using the canonical head lookup key.

// The fields below define the suffix of keys in the db.
var (
	proposedBlockBucket = []byte("proposed-block-bucket")
	attestationBucket   = []byte("attestation-bucket")
)

func getBucket(tx *bolt.Tx, pubKey *bls.PublicKey, forkVersion uint64, subBucket []byte) *bolt.Bucket {
	bucket := tx.Bucket(
		//reducing the nesting level of buckets, increased speed
		append(pubKey.Marshal(), bytesutil.Bytes8(forkVersion)...),
	)
	if bucket == nil {
		return nil
	}
	return bucket.Bucket(subBucket)
}

func (db *ValidatorDB) lastInAllForks(pubKey *bls.PublicKey, subBucket []byte, fn func([]byte) error) error {
	return db.db.View(func(tx *bolt.Tx) error {
		c := tx.Cursor()

		pubKeyBytes := pubKey.Marshal()
		for k, v := c.Seek(pubKeyBytes); k != nil && bytes.HasPrefix(k, pubKeyBytes); k, v = c.Next() {
			if v != nil {
				log.Debug("found value, not bucket")
				continue
			}
			bucket := tx.Bucket(k)
			subBucket := bucket.Bucket(proposedBlockBucket)

			// TODO test that the last() returns the maximum key, regardless of the order of push()
			_, lastInForkEnc := subBucket.Cursor().Last()
			if lastInForkEnc != nil {
				if err := fn(lastInForkEnc); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
