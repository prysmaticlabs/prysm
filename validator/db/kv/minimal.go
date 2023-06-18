package kv

import (
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	bolt "go.etcd.io/bbolt"
)

func isBucketMinimal(tx *bolt.Tx, bucket *bolt.Bucket, deep bool) bool {
	c := bucket.Cursor()

	k, _ := c.First()
	if k == nil {
		return true
	}

	firstSourceEpoch := bytesutil.BytesToEpochBigEndian(k)

	k, v := c.Last()
	lastSourceEpoch := bytesutil.BytesToEpochBigEndian(k)

	if firstSourceEpoch != lastSourceEpoch {
		return false
	}

	if deep {
		if len(v) > 8 {
			return false
		}
	}

	return true
}

func (s *Store) IsMinimal() (error, bool) {
	isMinimal := true

	err := s.view(func(tx *bolt.Tx) error {
		pubkeysBucket := tx.Bucket(pubKeysBucket)

		pubkeysBucket.ForEach(func(pubkey []byte, _ []byte) error {
			pubkeyBucket := pubkeysBucket.Bucket(pubkey)

			signingRootsBucket := pubkeyBucket.Bucket(attestationSigningRootsBucket)
			sourceEpochsBucket := pubkeyBucket.Bucket(attestationSourceEpochsBucket)
			targetEpochsBucket := pubkeyBucket.Bucket(attestationTargetEpochsBucket)

			if !(isBucketMinimal(tx, signingRootsBucket, false) &&
				isBucketMinimal(tx, sourceEpochsBucket, true) &&
				isBucketMinimal(tx, targetEpochsBucket, true)) {
				isMinimal = false
			}

			return nil
		})

		return nil
	})

	return err, isMinimal
}
