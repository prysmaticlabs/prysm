package kv

import (
	"errors"

	"github.com/boltdb/bolt"
	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/sirupsen/logrus"
)

var snappyKey = []byte("snappy")

func (kv *Store) ensureSnappy() error {
	var isMigrated bool

	kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(migrationBucket)
		v := bkt.Get(snappyKey)
		isMigrated = len(v) == 1 && v[0] == 0x01
		return nil
	})

	if !featureconfig.Get().EnableSnappyDBCompression {
		if isMigrated {
			return errors.New("beaconDB has been migrated to snappy compression, run with flag --snappy")
		}
		return nil
	}

	if isMigrated {
		return nil
	}

	log := logrus.WithField("prefix", "kv")
	log.Info("Compressing database to snappy compression. This might take a while...")

	bucketsToMigrate := [][]byte{
		attestationsBucket,
		blocksBucket,
		stateBucket,
		proposerSlashingsBucket,
		attesterSlashingsBucket,
		voluntaryExitsBucket,
		checkpointBucket,
		archivedValidatorSetChangesBucket,
		archivedCommitteeInfoBucket,
		archivedBalancesBucket,
		archivedValidatorParticipationBucket,
		finalizedBlockRootsIndexBucket,
	}

	return kv.db.Update(func(tx *bolt.Tx) error {
		for _, b := range bucketsToMigrate {
			log.WithField("bucket", string(b)).Debug("Compressing bucket.")
			if err := migrateBucketToSnappy(tx.Bucket(b)); err != nil {
				return err
			}
		}
		bkt := tx.Bucket(migrationBucket)
		return bkt.Put(snappyKey, []byte{0x01})
	})
}

func migrateBucketToSnappy(bkt *bolt.Bucket) error {
	c := bkt.Cursor()
	for key, val := c.First(); key != nil; key, val = c.Next() {
		if err := bkt.Put(key, snappy.Encode(nil, val)); err != nil {
			return err
		}
	}
	return nil
}
