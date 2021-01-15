package kv

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	bolt "go.etcd.io/bbolt"
)

func Test_migrateOptimalAttesterProtection(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, validatorDB *Store)
		eval  func(t *testing.T, validatorDB *Store)
	}{
		{
			name: "only runs once",
			setup: func(t *testing.T, validatorDB *Store) {
				err := validatorDB.update(func(tx *bolt.Tx) error {
					return tx.Bucket(migrationsBucket).Put(migrationOptimalAttesterProtectionKey, migrationCompleted)
				})
				require.NoError(t, err)
			},
			eval: func(t *testing.T, validatorDB *Store) {
				err := validatorDB.view(func(tx *bolt.Tx) error {
					data := tx.Bucket(migrationsBucket).Get(migrationOptimalAttesterProtectionKey)
					require.DeepEqual(t, data, migrationCompleted)
					return nil
				})
				require.NoError(t, err)
			},
		},
		{
			name: "populates optimized schema buckets",
			setup: func(t *testing.T, validatorDB *Store) {
				ctx := context.Background()
				pubKey := [48]byte{1}
				history := newDeprecatedAttestingHistory(0)
				// Attest all epochs from genesis to 50.
				numEpochs := uint64(50)
				for i := uint64(1); i <= numEpochs; i++ {
					var sr [32]byte
					copy(sr[:], fmt.Sprintf("%d", i))
					newHist, err := history.setTargetData(ctx, i, &deprecatedHistoryData{
						Source:      i - 1,
						SigningRoot: sr[:],
					})
					require.NoError(t, err)
					history = newHist
				}
				newHist, err := history.setLatestEpochWritten(ctx, numEpochs)
				require.NoError(t, err)

				err = validatorDB.update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(deprecatedAttestationHistoryBucket)
					return bucket.Put(pubKey[:], newHist)
				})
				require.NoError(t, err)
			},
			eval: func(t *testing.T, validatorDB *Store) {
				// Verify we indeed have the data for all epochs
				// since genesis to epoch 50 under the new schema.
				err := validatorDB.view(func(tx *bolt.Tx) error {
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
			setup: func(t *testing.T, validatorDB *Store) {
				ctx := context.Background()
				pubKey := [48]byte{1}
				history := newDeprecatedAttestingHistory(0)
				// Attest all epochs from genesis to 50.
				numEpochs := uint64(50)
				for i := uint64(1); i <= numEpochs; i++ {
					var sr [32]byte
					copy(sr[:], fmt.Sprintf("%d", i))
					newHist, err := history.setTargetData(ctx, i, &deprecatedHistoryData{
						Source:      i - 1,
						SigningRoot: sr[:],
					})
					require.NoError(t, err)
					history = newHist
				}
				newHist, err := history.setLatestEpochWritten(ctx, numEpochs)
				require.NoError(t, err)

				err = validatorDB.update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(deprecatedAttestationHistoryBucket)
					return bucket.Put(pubKey[:], newHist)
				})
				require.NoError(t, err)

				// Run the migration.
				require.NoError(t, validatorDB.migrateOptimalAttesterProtectionUp(ctx))

				// Then delete the migration completed key.
				err = validatorDB.update(func(tx *bolt.Tx) error {
					mb := tx.Bucket(migrationsBucket)
					return mb.Delete(migrationOptimalAttesterProtectionKey)
				})
				require.NoError(t, err)

				// Write one more entry to the DB with the old format.
				var sr [32]byte
				copy(sr[:], fmt.Sprintf("%d", numEpochs+1))
				newHist, err = newHist.setTargetData(ctx, numEpochs+1, &deprecatedHistoryData{
					Source:      numEpochs,
					SigningRoot: sr[:],
				})
				require.NoError(t, err)
				newHist, err = newHist.setLatestEpochWritten(ctx, numEpochs+1)
				require.NoError(t, err)

				err = validatorDB.update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(deprecatedAttestationHistoryBucket)
					return bucket.Put(pubKey[:], newHist)
				})
				require.NoError(t, err)
			},
			eval: func(t *testing.T, validatorDB *Store) {
				// Verify we indeed have the data for all epochs
				// since genesis to epoch 50+1 under the new schema.
				err := validatorDB.view(func(tx *bolt.Tx) error {
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
			validatorDB, err := setupDBWithoutMigration(t.TempDir())
			require.NoError(t, err, "Failed to instantiate DB")
			t.Cleanup(func() {
				require.NoError(t, validatorDB.Close(), "Failed to close database")
				require.NoError(t, validatorDB.ClearDB(), "Failed to clear database")
			})
			tt.setup(t, validatorDB)
			require.NoError(t, validatorDB.migrateOptimalAttesterProtectionUp(context.Background()))
			tt.eval(t, validatorDB)
		})
	}
}

func setupDBWithoutMigration(dirPath string) (*Store, error) {
	hasDir, err := fileutil.HasDir(dirPath)
	if err != nil {
		return nil, err
	}
	if !hasDir {
		if err := fileutil.MkdirAll(dirPath); err != nil {
			return nil, err
		}
	}
	datafile := filepath.Join(dirPath, ProtectionDbFileName)
	boltDB, err := bolt.Open(datafile, params.BeaconIoConfig().ReadWritePermissions, &bolt.Options{Timeout: params.BeaconIoConfig().BoltTimeout})
	if err != nil {
		if errors.Is(err, bolt.ErrTimeout) {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}

	kv := &Store{
		db:           boltDB,
		databasePath: dirPath,
	}

	if err := kv.db.Update(func(tx *bolt.Tx) error {
		return createBuckets(
			tx,
			genesisInfoBucket,
			deprecatedAttestationHistoryBucket,
			historicProposalsBucket,
			lowestSignedSourceBucket,
			lowestSignedTargetBucket,
			lowestSignedProposalsBucket,
			highestSignedProposalsBucket,
			pubKeysBucket,
			migrationsBucket,
		)
	}); err != nil {
		return nil, err
	}
	return kv, prometheus.Register(createBoltCollector(kv.db))
}
