package kv

import (
	"bytes"
	"context"

	"github.com/golang/snappy"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	statepb "github.com/prysmaticlabs/prysm/proto/prysm/v2/state"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	bolt "go.etcd.io/bbolt"
)

const (
	batchSize = 10
)

var migrationStateValidatorsKey = []byte("migration_state_validator")

type migrationRow struct {
	key   []byte
	value []byte
}

func migrateStateValidators(ctx context.Context, db *bolt.DB) error {
	migrateDB := false
	if updateErr := db.Update(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		// feature flag is not enabled
		// - migration is complete, don't migrate the DB but warn that this will work as if the flag is enabled.
		// - migration is not complete, don't migrate the DB.
		if !featureconfig.Get().EnableHistoricalSpaceRepresentation {
			b := mb.Get(migrationStateValidatorsKey)
			if bytes.Equal(b, migrationCompleted) {
				log.Warning("migration of historical states already completed. The node will work as if --enable-historical-state-representation=true.")
				return nil
			} else {
				return nil
			}
		}

		// if the migration flag is enabled (checked in the above condition)
		//  and if migration is complete, don't migrate again.
		if b := mb.Get(migrationStateValidatorsKey); bytes.Equal(b, migrationCompleted) {
			return nil
		}

		// migrate flag is enabled and DB is not migrated yet
		migrateDB = true
		return nil
	}); updateErr != nil {
		log.WithError(updateErr).Errorf("could not migrate bucket: %s", stateBucket)
		return updateErr
	}

	// do not migrate the DB
	if !migrateDB {
		return nil
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		stateBkt := tx.Bucket(stateBucket)
		if stateBkt == nil {
			return nil
		}
		valBkt := tx.Bucket(stateValidatorsBucket)
		if valBkt == nil {
			return nil
		}
		indexBkt := tx.Bucket(blockRootValidatorHashesBucket)
		if indexBkt == nil {
			return nil
		}
		mb := tx.Bucket(migrationsBucket)
		if mb == nil {
			return nil
		}
		doneC := make(chan bool)
		errC := make(chan error, batchSize)
		workC := make(chan *migrationRow, batchSize)
		go readStateEntriesFromBucket(ctx, stateBkt, workC, errC)
		go storeValidatorEntriesSeparately(ctx, stateBkt, valBkt, indexBkt, workC, doneC, errC)
		<-doneC
		return mb.Put(migrationStateValidatorsKey, migrationCompleted)
	}); err != nil {
		return err
	}
	return nil
}

func readStateEntriesFromBucket(ctx context.Context, stateBkt *bolt.Bucket, workC chan<- *migrationRow, errC chan error) {
	count := uint64(0)
	defer func() {
		close(workC)
	}()
	if forEachErr := stateBkt.ForEach(func(k, v []byte) error {
		row := &migrationRow{
			key:   k,
			value: v,
		}
		workC <- row
		count++

		select {
		case <-errC:
			break
		case <-ctx.Done():
			errC <- ctx.Err()
			break
		}
		return nil
	}); forEachErr != nil {
		log.WithError(forEachErr).Errorf("could not migrate row %d for bucket: %s", count, stateBucket)
		errC <- forEachErr
	}
}

func storeValidatorEntriesSeparately(ctx context.Context, stateBkt *bolt.Bucket, valBkt *bolt.Bucket, indexBkt *bolt.Bucket, workC <-chan *migrationRow, doneC chan<- bool, errC chan error) {
	defer func() {
		doneC <- true
	}()
	for mRow := range workC {
		state := &statepb.BeaconState{}
		if decodeErr := decode(ctx, mRow.value, state); decodeErr != nil {
			errC <- decodeErr
		}

		// move all the validators in this state registry out to a new bucket.
		var validatorKeys []byte
		for _, val := range state.Validators {
			valBytes, encodeErr := encode(ctx, val)
			if encodeErr != nil {
				errC <- encodeErr
			}

			// create the unique hash for that validator entry.
			hash := hashutil.Hash(valBytes)

			// add the validator in the stateValidatorsBucket, if it is not present.
			if valEntry := valBkt.Get(hash[:]); valEntry == nil {
				if putErr := valBkt.Put(hash[:], valBytes); putErr != nil {
					errC <- putErr
				}
			}

			// note down the pointer of the stateValidatorsBucket.
			validatorKeys = append(validatorKeys, hash[:]...)
		}

		// add the validator entry keys for a given block root.
		compValidatorKeys := snappy.Encode(nil, validatorKeys)
		idxErr := indexBkt.Put(mRow.key, compValidatorKeys)
		if idxErr != nil {
			errC <- idxErr
		}

		// zero the validator entries in BeaconState object .
		state.Validators = make([]*v1alpha1.Validator, 0)
		stateBytes, encodeErr := encode(ctx, state)
		if encodeErr != nil {
			errC <- encodeErr
		}
		if stateErr := stateBkt.Put(mRow.key, stateBytes); stateErr != nil {
			errC <- stateErr
		}
		select {
		case <-errC:
			break
		case <-ctx.Done():
			errC <- ctx.Err()
			break
		}
	}
}
