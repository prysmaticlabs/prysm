package kv

import (
	"context"
	"fmt"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	bolt "go.etcd.io/bbolt"
)

func TestStore_migrateSourceTargetEpochsBucketUp(t *testing.T) {
	numEpochs := uint64(100)
	numKeys := 50
	pubKeys := make([][48]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		var pk [48]byte
		copy(pk[:], fmt.Sprintf("%d", i))
		pubKeys[i] = pk
	}
	tests := []struct {
		name  string
		setup func(t *testing.T, validatorDB *Store)
		eval  func(t *testing.T, validatorDB *Store)
	}{
		{
			name: "only runs once",
			setup: func(t *testing.T, validatorDB *Store) {
				err := validatorDB.update(func(tx *bolt.Tx) error {
					return tx.Bucket(migrationsBucket).Put(migrationSourceTargetEpochsBucketKey, migrationCompleted)
				})
				require.NoError(t, err)
			},
			eval: func(t *testing.T, validatorDB *Store) {
				err := validatorDB.view(func(tx *bolt.Tx) error {
					data := tx.Bucket(migrationsBucket).Get(migrationSourceTargetEpochsBucketKey)
					require.DeepEqual(t, data, migrationCompleted)
					return nil
				})
				require.NoError(t, err)
			},
		},
		{
			name: "populates new target epochs bucket",
			setup: func(t *testing.T, validatorDB *Store) {
				err := validatorDB.update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(pubKeysBucket)
					for _, pubKey := range pubKeys {
						pkBucket, err := bucket.CreateBucketIfNotExists(pubKey[:])
						if err != nil {
							return err
						}
						sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
						if err != nil {
							return err
						}
						for epoch := uint64(1); epoch < numEpochs; epoch++ {
							source := epoch - 1
							target := epoch
							sourceEpoch := bytesutil.Uint64ToBytesBigEndian(source)
							targetEpoch := bytesutil.Uint64ToBytesBigEndian(target)
							if err := sourceEpochsBucket.Put(sourceEpoch, targetEpoch); err != nil {
								return err
							}
						}
					}
					return nil
				})
				require.NoError(t, err)
			},
			eval: func(t *testing.T, validatorDB *Store) {
				// Verify we indeed have the data for all epochs
				// since genesis to epoch 50 under the new schema.
				err := validatorDB.view(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(pubKeysBucket)
					for _, pubKey := range pubKeys {
						pkBucket := bucket.Bucket(pubKey[:])
						sourceEpochsBucket := pkBucket.Bucket(attestationSourceEpochsBucket)
						targetEpochsBucket := pkBucket.Bucket(attestationTargetEpochsBucket)

						// Verify we have (source epoch, target epoch) pairs.
						for sourceEpoch := uint64(0); sourceEpoch < numEpochs-1; sourceEpoch++ {
							sourceEpochBytes := bytesutil.Uint64ToBytesBigEndian(sourceEpoch)
							targetEpochBytes := sourceEpochsBucket.Get(sourceEpochBytes)
							targetEpoch := bytesutil.BytesToUint64BigEndian(targetEpochBytes)
							require.Equal(t, sourceEpoch+1, targetEpoch)
						}
						// Verify we have (target epoch, source epoch) pairs.
						for targetEpoch := uint64(1); targetEpoch < numEpochs; targetEpoch++ {
							targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(targetEpoch)
							sourceEpochBytes := targetEpochsBucket.Get(targetEpochBytes)
							sourceEpoch := bytesutil.BytesToUint64BigEndian(sourceEpochBytes)
							require.Equal(t, targetEpoch-1, sourceEpoch)
						}
					}
					return nil
				})
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatorDB := setupDB(t, pubKeys)
			tt.setup(t, validatorDB)
			require.NoError(t, validatorDB.migrateSourceTargetEpochsBucketUp(context.Background()))
			tt.eval(t, validatorDB)
		})
	}
}

func TestStore_migrateSourceTargetEpochsBucketDown(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, validatorDB *Store)
		eval  func(t *testing.T, validatorDB *Store)
	}{
		{
			name: "unsets the migration completed key upon completion",
			setup: func(t *testing.T, validatorDB *Store) {
				err := validatorDB.update(func(tx *bolt.Tx) error {
					return tx.Bucket(migrationsBucket).Put(migrationOptimalAttesterProtectionKey, migrationCompleted)
				})
				require.NoError(t, err)
			},
			eval: func(t *testing.T, validatorDB *Store) {
				err := validatorDB.view(func(tx *bolt.Tx) error {
					data := tx.Bucket(migrationsBucket).Get(migrationOptimalAttesterProtectionKey)
					require.DeepEqual(t, true, data == nil)
					return nil
				})
				require.NoError(t, err)
			},
		},
		{
			name:  "unsets the migration, even if unset already (no panic)",
			setup: func(t *testing.T, validatorDB *Store) {},
			eval: func(t *testing.T, validatorDB *Store) {
				// Ensure the migration is not marked as complete.
				err := validatorDB.view(func(tx *bolt.Tx) error {
					data := tx.Bucket(migrationsBucket).Get(migrationOptimalAttesterProtectionKey)
					require.DeepNotEqual(t, data, migrationCompleted)
					return nil
				})
				require.NoError(t, err)
			},
		},
		{
			name: "populates old format from data using the new schema",
			setup: func(t *testing.T, validatorDB *Store) {
				pubKeys := [][48]byte{{1}, {2}}
				// Create attesting history for two public keys
				err := validatorDB.update(func(tx *bolt.Tx) error {
					bkt := tx.Bucket(pubKeysBucket)
					for _, pubKey := range pubKeys {
						pkBucket, err := bkt.CreateBucketIfNotExists(pubKey[:])
						if err != nil {
							return err
						}
						sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
						if err != nil {
							return err
						}
						signingRootsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
						if err != nil {
							return err
						}
						// The highest epoch we write is 50.
						highestEpoch := uint64(50)
						for i := uint64(1); i <= highestEpoch; i++ {
							source := bytesutil.Uint64ToBytesBigEndian(i - 1)
							target := bytesutil.Uint64ToBytesBigEndian(i)
							if err := sourceEpochsBucket.Put(source, target); err != nil {
								return err
							}
							var signingRoot [32]byte
							copy(signingRoot[:], fmt.Sprintf("%d", target))
							if err := signingRootsBucket.Put(target, signingRoot[:]); err != nil {
								return err
							}
						}
					}
					// Finally, we mark the migration as completed to show that we have the
					// new, optimized format for attester protection in the database.
					migrationBkt := tx.Bucket(migrationsBucket)
					return migrationBkt.Put(migrationOptimalAttesterProtectionKey, migrationCompleted)
				})
				require.NoError(t, err)
			},
			eval: func(t *testing.T, validatorDB *Store) {
				ctx := context.Background()
				pubKeys := [][48]byte{{1}, {2}}
				// Next up, we validate that we have indeed rolled back our data
				// into the old format for attesting history.
				err := validatorDB.view(func(tx *bolt.Tx) error {
					bkt := tx.Bucket(deprecatedAttestationHistoryBucket)
					for _, pubKey := range pubKeys {
						encodedHistoryBytes := bkt.Get(pubKey[:])
						require.NotNil(t, encodedHistoryBytes)
						attestingHistory := deprecatedEncodedAttestingHistory(encodedHistoryBytes)
						highestEpoch, err := attestingHistory.getLatestEpochWritten(ctx)
						require.NoError(t, err)
						// Verify the highest epoch written is 50 from the setup stage.
						require.Equal(t, types.Epoch(50), highestEpoch)
					}
					return nil
				})
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatorDB := setupDB(t, nil)
			tt.setup(t, validatorDB)
			require.NoError(t, validatorDB.migrateSourceTargetEpochsBucketDown(context.Background()))
			tt.eval(t, validatorDB)
		})
	}
}
