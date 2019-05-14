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

func getBucket(tx *bolt.Tx, pubKey *bls.PublicKey, forkVersion uint64, subBucketKey []byte, createIfNotExists bool) *bolt.Bucket {
	//reducing the nesting level of buckets, increased speed
	parentBucketKey := append(pubKey.Marshal(), bytesutil.Bytes8(forkVersion)...)
	parentBucket := tx.Bucket(parentBucketKey)
	if parentBucket == nil {
		if createIfNotExists {
			var err error
			parentBucket, err = tx.CreateBucket(parentBucketKey)
			if err != nil {
				log.WithError(err).Error("don't can create bucket")
				return nil
			}
		} else {
			return nil
		}
	}
	subBucket := parentBucket.Bucket(subBucketKey)
	if subBucket == nil && createIfNotExists {
		var err error
		subBucket, err = parentBucket.CreateBucket(subBucketKey)
		if err != nil {
			log.WithError(err).Error("don't can create bucket")
			return nil
		}
	}
	return subBucket
}

func (db *ValidatorDB) lastInAllForks(pubKey *bls.PublicKey, subBucketKey []byte, fn func([]byte, []byte) error) error {
	return db.db.View(func(tx *bolt.Tx) error {
		c := tx.Cursor()

		pubKeyBytes := pubKey.Marshal()
		var maxKey, valueForMaxKey []byte
		for k, v := c.Seek(pubKeyBytes); k != nil && bytes.HasPrefix(k, pubKeyBytes); k, v = c.Next() {
			if v != nil {
				log.Debug("found value, not bucket")
				continue
			}
			bucket := tx.Bucket(k)
			subBucket := bucket.Bucket(subBucketKey)

			key, value := subBucket.Cursor().Last()
			if bytes.Compare(maxKey, key) < 0 {
				maxKey = key
				valueForMaxKey = value
			}

		}
		return fn(maxKey, valueForMaxKey)
	})
}
