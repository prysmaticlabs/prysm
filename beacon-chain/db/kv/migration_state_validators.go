package kv

import (
	"bytes"
	"context"

	"github.com/golang/snappy"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/progressutil"
	bolt "go.etcd.io/bbolt"
)

var migrationStateValidatorsKey = []byte("migration_state_validator")

func migrateStateValidators(tx *bolt.Tx) error {
	mb := tx.Bucket(migrationsBucket)
	if !featureconfig.Get().EnableHistoricalSpaceRepresentation {
		if b := mb.Get(migrationStateValidatorsKey); bytes.Equal(b, migrationCompleted) {
			log.Warning("migration of historical states already completed. The node will work as if --enable-historical-state-representation=true.")
			return nil // Migration already completed.
		}
	}

	// if the flag is enabled and migration is completed, dont migrate again.
	if b := mb.Get(migrationStateValidatorsKey); bytes.Equal(b, migrationCompleted) {
		return nil
	}
	stateBkt := tx.Bucket(stateBucket)
	if stateBkt == nil {
		return nil
	}

	// get the count of keys in the state bucket for passing it to the progress indicator.
	count, err := stateCount(stateBkt)
	if err != nil {
		return err
	}

	// get the source and destination buckets.
	log.Infof("Performing a one-time migration to a more efficient database schema for %s. It will take few minutes", stateBucket)
	bar := progressutil.InitializeProgressBar(count, "Migrating state validators to new schema.")

	valBkt := tx.Bucket(stateValidatorsBucket)
	if valBkt == nil {
		return nil
	}
	indexBkt := tx.Bucket(blockRootValidatorHashesBucket)
	if indexBkt == nil {
		return nil
	}

	// for each of the state in the stateBucket, do the migration.
	ctx := context.Background()
	c := stateBkt.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		state := &statepb.BeaconState{}
		if decodeErr := decode(ctx, v, state); decodeErr != nil {
			return decodeErr
		}

		// move all the validators in this state registry out to a new bucket.
		var validatorKeys []byte
		for _, val := range state.Validators {
			valBytes, encodeErr := encode(ctx, val)
			if encodeErr != nil {
				return encodeErr
			}

			// create the unique hash for that validator entry.
			hash := hashutil.Hash(valBytes)

			// add the validator in the stateValidatorsBucket, if it is not present.
			if valEntry := valBkt.Get(hash[:]); valEntry == nil {
				if putErr := valBkt.Put(hash[:], valBytes); putErr != nil {
					return putErr
				}
			}

			// note down the pointer of the stateValidatorsBucket.
			validatorKeys = append(validatorKeys, hash[:]...)
		}

		// add the validator entry keys for a given block root.
		compValidatorKeys := snappy.Encode(nil, validatorKeys)
		idxErr := indexBkt.Put(k, compValidatorKeys)
		if idxErr != nil {
			return idxErr
		}

		// zero the validator entries in BeaconState object .
		state.Validators = make([]*v1alpha1.Validator, 0)
		stateBytes, encodeErr := encode(ctx, state)
		if encodeErr != nil {
			return encodeErr
		}
		if stateErr := stateBkt.Put(k, stateBytes); stateErr != nil {
			return stateErr
		}
		if barErr := bar.Add(1); barErr != nil {
			return barErr
		}
	}

	// Mark migration complete.
	return mb.Put(migrationStateValidatorsKey, migrationCompleted)
}

func stateCount(stateBucket *bolt.Bucket) (int, error) {
	count := 0
	if err := stateBucket.ForEach(func(pubKey, v []byte) error {
		count++
		return nil
	}); err != nil {
		return 0, nil
	}
	return count, nil
}
