package kv

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/snappy"
	bolt "go.etcd.io/bbolt"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestMemory_migrateOptimalAttesterProtection(t *testing.T) {
	ctx := context.Background()
	numValidators := 500
	pubKeys := make([][48]byte, numValidators)
	for i := 0; i < numValidators; i++ {
		var pk [48]byte
		copy(pk[:], fmt.Sprintf("%d", i))
		pubKeys[i] = pk
	}
	validatorDB := setupDB(t, pubKeys)
	history := NewAttestationHistoryArray(0)
	// Attest all epochs from genesis to 8500 (similar to mainnet).
	numEpochs := uint64(8500)
	for i := uint64(1); i <= numEpochs; i++ {
		var sr [32]byte
		copy(sr[:], fmt.Sprintf("%d", i))
		newHist, err := history.SetTargetData(ctx, i, &HistoryData{
			Source:      i - 1,
			SigningRoot: sr[:],
		})
		require.NoError(t, err)
		history = newHist
	}
	newHist, err := history.SetLatestEpochWritten(ctx, numEpochs)
	require.NoError(t, err)
	enc := snappy.Encode(nil /*dst*/, newHist)

	err = validatorDB.setupHistoryForTest(pubKeys, enc)
	require.NoError(t, err)

	// Attempt a migration.
	fmt.Printf("Attempting migration for %d validators each attesting %d epochs\n", numValidators, numEpochs)
	start := time.Now()
	require.NoError(t, validatorDB.migrateTxCommit())
	end := time.Now()
	fmt.Printf("Migration complete, took %v\n", end.Sub(start))
}

func Test_migrateOptimalAttesterProtection(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, db *bolt.DB)
		eval  func(t *testing.T, db *bolt.DB)
	}{
		{
			name: "only runs once",
			setup: func(t *testing.T, db *bolt.DB) {
				err := db.Update(func(tx *bolt.Tx) error {
					return tx.Bucket(migrationsBucket).Put(migrationOptimalAttesterProtectionKey, migrationCompleted)
				})
				require.NoError(t, err)
			},
			eval: func(t *testing.T, db *bolt.DB) {
				err := db.View(func(tx *bolt.Tx) error {
					data := tx.Bucket(migrationsBucket).Get(migrationOptimalAttesterProtectionKey)
					require.DeepEqual(t, data, migrationCompleted)
					return nil
				})
				require.NoError(t, err)
			},
		},
		{
			name: "populates optimized schema buckets",
			setup: func(t *testing.T, db *bolt.DB) {
				ctx := context.Background()
				pubKey := [48]byte{1}
				history := NewAttestationHistoryArray(0)
				// Attest all epochs from genesis to 50.
				numEpochs := uint64(50)
				for i := uint64(1); i <= numEpochs; i++ {
					var sr [32]byte
					copy(sr[:], fmt.Sprintf("%d", i))
					newHist, err := history.SetTargetData(ctx, i, &HistoryData{
						Source:      i - 1,
						SigningRoot: sr[:],
					})
					require.NoError(t, err)
					history = newHist
				}
				newHist, err := history.SetLatestEpochWritten(ctx, numEpochs)
				require.NoError(t, err)

				err = db.Update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(historicAttestationsBucket)
					enc := snappy.Encode(nil /*dst*/, newHist)
					return bucket.Put(pubKey[:], enc)
				})
				require.NoError(t, err)
			},
			eval: func(t *testing.T, db *bolt.DB) {
				// Verify we indeed have the data for all epochs
				// since genesis to epoch 50 under the new schema.
				err := db.View(func(tx *bolt.Tx) error {
					pubKey := [48]byte{1}
					bucket := tx.Bucket(pubKeysBucket)
					pkBucket := bucket.Bucket(pubKey[:])
					signingRootsBucket := pkBucket.Bucket(attestationSigningRootsBucket)
					sourceEpochsBucket := pkBucket.Bucket(attestationSourceEpochsBucket)
					numEpochs := uint64(50)

					// Verify we have signing roots for target epochs 1 to 50 correctly.
					for targetEpoch := uint64(1); targetEpoch <= numEpochs; targetEpoch++ {
						var sr [32]byte
						copy(sr[:], fmt.Sprintf("%d", targetEpoch))
						targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(targetEpoch)
						migratedSigningRoot := signingRootsBucket.Get(targetEpochBytes)
						require.DeepEqual(t, sr[:], migratedSigningRoot)
					}

					// Verify we have (source epoch, target epoch) pairs for epochs 0 to 50 correctly.
					for sourceEpoch := uint64(0); sourceEpoch < numEpochs; sourceEpoch++ {
						sourceEpochBytes := bytesutil.Uint64ToBytesBigEndian(sourceEpoch)
						targetEpochBytes := sourceEpochsBucket.Get(sourceEpochBytes)
						targetEpoch := bytesutil.BytesToUint64BigEndian(targetEpochBytes)
						require.Equal(t, sourceEpoch+1, targetEpoch)
					}
					return nil
				})
				require.NoError(t, err)
			},
		},
		{
			name: "partial data saved for both types still completes the migration successfully",
			setup: func(t *testing.T, db *bolt.DB) {
				ctx := context.Background()
				pubKey := [48]byte{1}
				history := NewAttestationHistoryArray(0)
				// Attest all epochs from genesis to 50.
				numEpochs := uint64(50)
				for i := uint64(1); i <= numEpochs; i++ {
					var sr [32]byte
					copy(sr[:], fmt.Sprintf("%d", i))
					newHist, err := history.SetTargetData(ctx, i, &HistoryData{
						Source:      i - 1,
						SigningRoot: sr[:],
					})
					require.NoError(t, err)
					history = newHist
				}
				newHist, err := history.SetLatestEpochWritten(ctx, numEpochs)
				require.NoError(t, err)

				err = db.Update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(historicAttestationsBucket)
					enc := snappy.Encode(nil /*dst*/, newHist)
					return bucket.Put(pubKey[:], enc)
				})
				require.NoError(t, err)

				// Run the migration.
				require.NoError(t, db.Update(migrateOptimalAttesterProtection))

				// Then delete the migration completed key.
				err = db.Update(func(tx *bolt.Tx) error {
					mb := tx.Bucket(migrationsBucket)
					return mb.Delete(migrationOptimalAttesterProtectionKey)
				})
				require.NoError(t, err)

				// Write one more entry to the DB with the old format.
				var sr [32]byte
				copy(sr[:], fmt.Sprintf("%d", numEpochs+1))
				newHist, err = newHist.SetTargetData(ctx, numEpochs+1, &HistoryData{
					Source:      numEpochs,
					SigningRoot: sr[:],
				})
				require.NoError(t, err)
				newHist, err = newHist.SetLatestEpochWritten(ctx, numEpochs+1)
				require.NoError(t, err)

				err = db.Update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(historicAttestationsBucket)
					enc := snappy.Encode(nil /*dst*/, newHist)
					return bucket.Put(pubKey[:], enc)
				})
				require.NoError(t, err)
			},
			eval: func(t *testing.T, db *bolt.DB) {
				// Verify we indeed have the data for all epochs
				// since genesis to epoch 50+1 under the new schema.
				err := db.View(func(tx *bolt.Tx) error {
					pubKey := [48]byte{1}
					bucket := tx.Bucket(pubKeysBucket)
					pkBucket := bucket.Bucket(pubKey[:])
					signingRootsBucket := pkBucket.Bucket(attestationSigningRootsBucket)
					sourceEpochsBucket := pkBucket.Bucket(attestationSourceEpochsBucket)
					numEpochs := uint64(50)

					// Verify we have signing roots for target epochs 1 to 50 correctly.
					for targetEpoch := uint64(1); targetEpoch <= numEpochs+1; targetEpoch++ {
						var sr [32]byte
						copy(sr[:], fmt.Sprintf("%d", targetEpoch))
						targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(targetEpoch)
						migratedSigningRoot := signingRootsBucket.Get(targetEpochBytes)
						require.DeepEqual(t, sr[:], migratedSigningRoot)
					}

					// Verify we have (source epoch, target epoch) pairs for epochs 0 to 50 correctly.
					for sourceEpoch := uint64(0); sourceEpoch < numEpochs+1; sourceEpoch++ {
						sourceEpochBytes := bytesutil.Uint64ToBytesBigEndian(sourceEpoch)
						targetEpochBytes := sourceEpochsBucket.Get(sourceEpochBytes)
						targetEpoch := bytesutil.BytesToUint64BigEndian(targetEpochBytes)
						require.Equal(t, sourceEpoch+1, targetEpoch)
					}
					return nil
				})
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t, nil).db
			tt.setup(t, db)
			assert.NoError(t, db.Update(migrateOptimalAttesterProtection), "migrateOptimalAttesterProtection(tx) error")
			tt.eval(t, db)
		})
	}
}
