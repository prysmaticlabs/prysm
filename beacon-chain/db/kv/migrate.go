package kv

import (
	"bytes"
	"fmt"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/monitoring/progress"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/urfave/cli/v2"
)

const batchSize = 10

var migrationStateKey = []byte("migration_state")

// MigrateStateDB normalizes the state object and saves space.
// it also saves space by not storing the already stored elements in the
// circular buffers.
func MigrateStateDB(cliCtx *cli.Context) error {
	dataDir := cliCtx.String(cmd.DataDirFlag.Name)
	dbDir := path.Join(dataDir, BeaconNodeDbDirName)
	dbNameWithPath := path.Join(dbDir, DatabaseFileName)

	// check if the database file exists
	if !file.FileExists(dbNameWithPath) {
		return errors.New(fmt.Sprintf("database file not found: %s", dbNameWithPath))
	}

	// open the raw database file. If the file is busy, then exit.
	db, openErr := bolt.Open(dbNameWithPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if openErr != nil {
		return errors.New(fmt.Sprintf("could not open db: , %v", openErr))
	}



	yes, valErr := isValidatorEntriesAlreadyMigrated(db)
	if valErr != nil {
		return errors.New(fmt.Sprintf("could not check if entries migrated : , %v", valErr))
	}

	if yes {
		migrateWithValidatorEntries(db)
	} else {
		migrateOnlyTheValidatorEntryHashes(db)
	}

	log.Info("migration completed successfully")
	return nil
}

func isValidatorEntriesAlreadyMigrated(db *bolt.DB) (bool, error) {
	// if the flag is not enabled, but the migration is over, then
	// follow the new code path as if the flag is enabled.
	returnFlag := false
	if err := db.View(func(tx *bolt.Tx) error {
		mb := tx.Bucket(migrationsBucket)
		b := mb.Get(migrationStateValidatorsKey)
		returnFlag = bytes.Equal(b, migrationCompleted)
		return nil
	}); err != nil {
		return returnFlag, err
	}
	return returnFlag, nil
}

func migrateWithValidatorEntries(db *bolt.DB) error {
	// create the new buckets
	// - stateValidatorsBucket
	// - stateValidatorsHashesBucket
	if err := createNewBuckets(db, stateValidatorsBucket, stateValidatorHashesBucket); err != nil {
		return err
	}

	// get the state bucket row count.
	// this is used to show the progress bar.
	keys, err := getStateBucketKeys(db)
	if err != nil{
		return err
	}

	// prepare the progress bar with the total count of the keys to migrate
	bar := progress.InitializeProgressBar(len(keys), "Migrating state bucket to new schema.")


	// for each state, do in batches
	// - store the unique validator entries in stateValidatorsBucket
	// - get the ID for the validator entrypoint
	// compress all the validator entry IDs and store in the "validators" field in state
	batchNo := 0
	for batchIndex := 0; batchIndex < len(keys); batchIndex += batchSize {
		if err := db.Update(func(tx *bolt.Tx) error {
			//create the source and destination buckets
			stateBkt := tx.Bucket(stateBucket);
			if stateBkt == nil {
				return errors.New("could not open \"state\" bucket")
			}
			valEntryBkt := tx.Bucket(stateValidatorsBucket)
			if valEntryBkt == nil {
				return errors.New("could not open \"state-validators\" bucket")
			}
			valHashBkt := tx.Bucket(stateValidatorHashesBucket)
			if valHashBkt == nil {
				return errors.New("could not open \"state-validator-hashes\" bucket")
			}

			// migrate the key values for this batch
			cursor := stateBkt.Cursor()
			count := 0
			index := batchIndex
			for _, v := cursor.Seek(keys[index]); count < batchSize && index < len(keys); _, v = cursor.Next() {
				enc, err := snappy.Decode(nil, v)
				if err != nil {
					return err
				}

					protoState := &v1alpha1.BeaconState{}
					if err := protoState.UnmarshalSSZ(enc); err != nil {
						return errors.Wrap(err, "failed to unmarshal encoding for phase0")
					}
					// no validators in state to migrate
					if len(protoState.Validators) == 0 {
						return fmt.Errorf("no validator entries in state key 0x%s", hexutil.Encode(keys[index]))
					}


					validatorHashes, err := insertValidators(ctx, protoState.Validators, valEntryBkt)
					if err != nil {
						return err
					}

					validaorIndexs, err := insertValidatorHashes(ctx, validatorHashes)
					if err != nil {
						return err
					}

					// add the validator entry indexes for a given block root.
					compValidatorIndexs := snappy.Encode(nil, validaorIndexs)


				// zero the validator entries in BeaconState object .
				protoState.Validators = make([]*v1alpha1.Validator, 0)
				rawObj, err := protoState.MarshalSSZ()
				if err != nil {
					return err
				}
				stateBytes := snappy.Encode(nil, append(altairKey, rawObj...))
				if stateErr := stateBkt.Put(keys[index], stateBytes); stateErr != nil {
					return stateErr
				}


					// add the entry indexs in validator field of BeaconState object .
					protoState.Validators = compValidatorIndexs
					rawObj, err := protoState.MarshalSSZ()
					if err != nil {
					return err
				}
					stateBytes, err := encode(ctx, compValidatorIndexs)
					if err != nil {
						return err
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
		return mb.Put(migrationStateKey, migrationCompleted)
	}); err != nil {
		return err
	}
	log.Infof("migration done for bucket %s.", stateBucket)
	return nil

}

func migrateOnlyTheValidatorEntryHashes(db *bolt.DB) {

}

func createNewBuckets(db *bolt.DB, buckets ...[]byte) error {
	if err := db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range buckets {
			if _, err := tx.CreateBucket(bucket); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func getStateBucketKeys(db *bolt.DB) ([][]byte, error){
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
		return nil, err
	}
	return keys, nil
}