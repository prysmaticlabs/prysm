package kv

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/snappy"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/progressutil"
	bolt "go.etcd.io/bbolt"
)

const batchSize = 10

var migrationStateValidatorsKey = []byte("migration_state_validator")

func migrateStateValidators(ctx context.Context, db *bolt.DB) error {
	migrateDB := false
	if updateErr := db.View(func(tx *bolt.Tx) error {
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

	log.Infof("Performing a one-time migration to a more efficient database schema for %s. It will take few minutes", stateBucket)

	// get all the keys to migrate
	var keys [][]byte
	if err := db.Update(func(tx *bolt.Tx) error {
		stateBkt := tx.Bucket(stateBucket)
		if stateBkt == nil {
			return nil
		}
		k, err := stateBucketKeys(stateBkt)
		if err != nil {
			return err
		}
		keys = k
		return nil
	}); err != nil {
		return err
	}
	log.Infof("total keys = %d", len(keys))

	// prepare the progress bar with the total count of the keys to migrate
	bar := progressutil.InitializeProgressBar(len(keys), "Migrating state validators to new schema.")

	batchNo := 0
	for batchIndex := 0; batchIndex < len(keys); batchIndex += batchSize {
		if err := db.Update(func(tx *bolt.Tx) error {
			//create the source and destination buckets
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

			// migrate the key values for this batch
			cursor := stateBkt.Cursor()
			count := 0
			index := batchIndex
			for _, v := cursor.Seek(keys[index]); count < batchSize && index < len(keys); _, v = cursor.Next() {
				state := &v1alpha1.BeaconState{}
				if decodeErr := decode(ctx, v, state); decodeErr != nil {
					return decodeErr
				}

				// no validators in state to migrate
				if len(state.Validators) == 0 {
					return errors.New(fmt.Sprintf("no validator entries in state 0x%s", hexutil.Encode(keys[index])))
				}

				// move all the validators in this state registry out to a new bucket.
				var validatorKeys []byte
				for _, val := range state.Validators {
					valBytes, encodeErr := encode(ctx, val)
					if encodeErr != nil {
						return encodeErr
					}

					// create the unique hash for that validator entry.
					hash, hashErr := val.HashTreeRoot()
					if hashErr != nil {
						return hashErr
					}

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
				idxErr := indexBkt.Put(keys[index], compValidatorKeys)
				if idxErr != nil {
					return idxErr
				}

				// zero the validator entries in BeaconState object .
				state.Validators = make([]*v1alpha1.Validator, 0)
				stateBytes, encodeErr := encode(ctx, state)
				if encodeErr != nil {
					return encodeErr
				}
				if stateErr := stateBkt.Put(keys[index], stateBytes); stateErr != nil {
					return stateErr
				}
				count++
				index++

				if barErr := bar.Add(1); barErr != nil {
					return barErr
				}
			}

			return nil
		}); err != nil {
			return err
		}
		batchNo++
	}

	// set the migration entry to done
	if err := db.Update(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		if mb == nil {
			return nil
		}
		return mb.Put(migrationStateValidatorsKey, migrationCompleted)
	}); err != nil {
		return err
	}
	log.Infof("migration done for bucket %s.", stateBucket)
	return nil
}

func stateBucketKeys(stateBucket *bolt.Bucket) ([][]byte, error) {
	var keys [][]byte
	if err := stateBucket.ForEach(func(pubKey, v []byte) error {
		keys = append(keys, pubKey)
		return nil
	}); err != nil {
		return nil, err
	}
	return keys, nil
}
