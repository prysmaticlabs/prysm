package kv

import (
	"context"
	"fmt"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	bolt "go.etcd.io/bbolt"
)

func TestPruneAttestationsOlderThanCurrentWeakSubjectivity_BeforeWeakSubjectivity_NoPruning(t *testing.T) {
	numEpochs := params.BeaconConfig().WeakSubjectivityPeriod
	pubKey := [48]byte{1}
	validatorDB := setupDB(t, [][48]byte{pubKey})

	// Write attesting history for every single epoch
	// since genesis to WEAK_SUBJECTIVITY_PERIOD.
	err := setupAttestationsForEveryEpoch(t, validatorDB, pubKey, numEpochs)
	require.NoError(t, err)

	// Next, attempt to prune and realize that we still have all epochs intact
	// because the highest epoch we have written is still within the
	// weak subjectivity period.
	err = validatorDB.PruneAttestationsOlderThanCurrentWeakSubjectivity(context.Background())
	require.NoError(t, err)

	err = checkAttestingHistoryAfterPruning(
		t, validatorDB, pubKey, numEpochs, false, /* should be pruned */
	)
	require.NoError(t, err)
}

func TestPruneAttestationsOlderThanCurrentWeakSubjectivity_AfterFirstWeakSubjectivity(t *testing.T) {
	numEpochs := params.BeaconConfig().WeakSubjectivityPeriod
	pubKey := [48]byte{1}
	validatorDB := setupDB(t, [][48]byte{pubKey})

	// Write attesting history for every single epoch
	// since genesis to WEAK_SUBJECTIVITY_PERIOD.
	err := setupAttestationsForEveryEpoch(t, validatorDB, pubKey, numEpochs)
	require.NoError(t, err)

	// Save a single attestation for WEAK_SUBJECTIVITY_PERIOD+1
	err = validatorDB.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBucket := bucket.Bucket(pubKey[:])
		sourceEpochsBkt := pkBucket.Bucket(attestationSourceEpochsBucket)
		signingRootsBkt := pkBucket.Bucket(attestationSigningRootsBucket)
		targetEpochBytes := bytesutil.EpochToBytesBigEndian(numEpochs + 1)
		sourceEpochBytes := bytesutil.EpochToBytesBigEndian(numEpochs)
		if err := sourceEpochsBkt.Put(sourceEpochBytes, targetEpochBytes); err != nil {
			return err
		}

		var signingRoot [32]byte
		copy(signingRoot[:], fmt.Sprintf("%d", targetEpochBytes))
		return signingRootsBkt.Put(targetEpochBytes, signingRoot[:])
	})
	require.NoError(t, err)

	err = validatorDB.PruneAttestationsOlderThanCurrentWeakSubjectivity(context.Background())
	require.NoError(t, err)

	// Next, attempt to prune and realize that we pruned everything except for
	// a signing root at target = WEAK_SUBJECTIVITY_PERIOD + 1.
	err = checkAttestingHistoryAfterPruning(
		t, validatorDB, pubKey, numEpochs, true, /* should be pruned */
	)
	require.NoError(t, err)
	err = validatorDB.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBucket := bucket.Bucket(pubKey[:])
		sourceEpochsBkt := pkBucket.Bucket(attestationSourceEpochsBucket)
		signingRootsBkt := pkBucket.Bucket(attestationSigningRootsBucket)

		targetEpochBytes := bytesutil.EpochToBytesBigEndian(numEpochs + 1)
		sourceEpochBytes := bytesutil.EpochToBytesBigEndian(numEpochs)

		storedTargetEpoch := sourceEpochsBkt.Get(sourceEpochBytes)
		require.DeepEqual(t, numEpochs+1, bytesutil.BytesToEpochBigEndian(storedTargetEpoch))

		var expectedSigningRoot [32]byte
		copy(expectedSigningRoot[:], fmt.Sprintf("%d", targetEpochBytes))
		signingRoot := signingRootsBkt.Get(targetEpochBytes)

		// Expect the correct signing root at WEAK_SUBJECTIVITY_PERIOD + 1.
		require.DeepEqual(t, expectedSigningRoot[:], signingRoot)
		return nil
	})
	require.NoError(t, err)
}

func TestPruneAttestationsOlderThanCurrentWeakSubjectivity_AfterMultipleWeakSubjectivity(t *testing.T) {
	numWeakSubjectivityPeriods := types.Epoch(5)
	pubKeys := [][48]byte{{1}}
	validatorDB := setupDB(t, pubKeys)

	// Write signing roots for epochs within multiples of WEAK_SUBJECTIVITY_PERIOD.
	err := validatorDB.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBucket, err := bucket.CreateBucketIfNotExists(pubKeys[0][:])
		if err != nil {
			return err
		}
		sourceEpochsBkt, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
		if err != nil {
			return err
		}
		signingRootsBkt, err := pkBucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
		if err != nil {
			return err
		}
		for i := types.Epoch(1); i <= numWeakSubjectivityPeriods; i++ {
			targetEpoch := (i * params.BeaconConfig().WeakSubjectivityPeriod) + 1
			targetEpochBytes := bytesutil.EpochToBytesBigEndian(targetEpoch)
			sourceEpoch := targetEpoch - 1
			sourceEpochBytes := bytesutil.EpochToBytesBigEndian(sourceEpoch)
			if err := sourceEpochsBkt.Put(sourceEpochBytes, targetEpochBytes); err != nil {
				return err
			}

			var signingRoot [32]byte
			copy(signingRoot[:], fmt.Sprintf("%d", targetEpochBytes))
			if err := signingRootsBkt.Put(targetEpochBytes, signingRoot[:]); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	err = validatorDB.PruneAttestationsOlderThanCurrentWeakSubjectivity(context.Background())
	require.NoError(t, err)

	// Next, attempt to prune and realize that we pruned everything except for
	// a signing root in an epoch within the highest WEAK_SUBJECTIVITY_PERIOD
	// multiple we wrote earlier in our test.
	err = validatorDB.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBucket := bucket.Bucket(pubKeys[0][:])
		signingRootsBkt := pkBucket.Bucket(attestationSigningRootsBucket)
		sourceEpochsBkt := pkBucket.Bucket(attestationSourceEpochsBucket)

		// We check everything except for the highest weak subjectivity period
		// has been pruned from the bucket.
		for i := types.Epoch(1); i <= numWeakSubjectivityPeriods-1; i++ {
			targetEpoch := (i * params.BeaconConfig().WeakSubjectivityPeriod) + 1
			sourceEpoch := targetEpoch - 1
			sourceEpochBytes := bytesutil.EpochToBytesBigEndian(sourceEpoch)
			targetEpochBytes := bytesutil.EpochToBytesBigEndian(targetEpoch)

			storedTargetEpoch := sourceEpochsBkt.Get(sourceEpochBytes)
			signingRoot := signingRootsBkt.Get(targetEpochBytes)
			// We expect to have pruned all these signing roots and epochs.
			require.Equal(t, true, signingRoot == nil)
			require.Equal(t, true, storedTargetEpoch == nil)
		}

		targetEpoch := (numWeakSubjectivityPeriods * params.BeaconConfig().WeakSubjectivityPeriod) + 1
		sourceEpoch := targetEpoch - 1
		targetEpochBytes := bytesutil.EpochToBytesBigEndian(targetEpoch)
		sourceEpochBytes := bytesutil.EpochToBytesBigEndian(sourceEpoch)
		var expectedSigningRoot [32]byte
		copy(expectedSigningRoot[:], fmt.Sprintf("%d", targetEpochBytes))
		signingRoot := signingRootsBkt.Get(targetEpochBytes)
		storedTargetEpoch := sourceEpochsBkt.Get(sourceEpochBytes)

		// Expect the correct signing root and target epoch.
		require.DeepEqual(t, expectedSigningRoot[:], signingRoot)
		require.DeepEqual(t, targetEpochBytes, storedTargetEpoch)
		return nil
	})
	require.NoError(t, err)
}

// Saves attesting history for every (source, target = source + 1) pairs since genesis
// up to a given number of epochs for a validator public key.
func setupAttestationsForEveryEpoch(t testing.TB, validatorDB *Store, pubKey [48]byte, numEpochs types.Epoch) error {
	return validatorDB.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBucket, err := bucket.CreateBucketIfNotExists(pubKey[:])
		if err != nil {
			return err
		}
		signingRootsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
		if err != nil {
			return err
		}
		sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
		if err != nil {
			return err
		}
		for targetEpoch := types.Epoch(1); targetEpoch < numEpochs; targetEpoch++ {
			targetEpochBytes := bytesutil.EpochToBytesBigEndian(targetEpoch)
			sourceEpochBytes := bytesutil.EpochToBytesBigEndian(targetEpoch - 1)
			// Save (source epoch, target epoch) pairs.
			if err := sourceEpochsBucket.Put(sourceEpochBytes, targetEpochBytes); err != nil {
				return err
			}
			// Save signing root for target epoch.
			var signingRoot [32]byte
			copy(signingRoot[:], fmt.Sprintf("%d", targetEpochBytes))
			if err := signingRootsBucket.Put(targetEpochBytes, signingRoot[:]); err != nil {
				return err
			}
		}
		return nil
	})
}

// Verifies, based on a boolean input argument, whether or not we should have
// pruned all attesting history since genesis up to a specified number of epochs.
func checkAttestingHistoryAfterPruning(
	t testing.TB, validatorDB *Store, pubKey [48]byte, numEpochs types.Epoch, shouldBePruned bool,
) error {
	return validatorDB.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBkt := bucket.Bucket(pubKey[:])
		signingRootsBkt := pkBkt.Bucket(attestationSigningRootsBucket)
		sourceEpochsBkt := pkBkt.Bucket(attestationSourceEpochsBucket)
		for targetEpoch := types.Epoch(1); targetEpoch < numEpochs; targetEpoch++ {
			targetEpochBytes := bytesutil.EpochToBytesBigEndian(targetEpoch)
			sourceEpochBytes := bytesutil.EpochToBytesBigEndian(targetEpoch - 1)

			storedTargetEpoch := sourceEpochsBkt.Get(sourceEpochBytes)
			signingRoot := signingRootsBkt.Get(targetEpochBytes)
			if shouldBePruned {
				// Expect to have no data if we have pruned.
				require.Equal(t, true, signingRoot == nil)
				require.Equal(t, true, storedTargetEpoch == nil)
			} else {
				// Expect the correct signing root.
				var expectedSigningRoot [32]byte
				copy(expectedSigningRoot[:], fmt.Sprintf("%d", targetEpochBytes))
				require.DeepEqual(t, expectedSigningRoot[:], signingRoot)
				// Expect the correct target epoch.
				require.DeepEqual(t, targetEpochBytes, storedTargetEpoch)
			}
		}
		return nil
	})
}

func Test_olderThanCurrentWeakSubjectivityPeriod(t *testing.T) {
	wssPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	type args struct {
		epoch        types.Epoch
		highestEpoch types.Epoch
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "returns true if epoch < WSS",
			args: args{
				epoch:        1,
				highestEpoch: wssPeriod,
			},
			want: true,
		},
		{
			name: "returns true if epoch == WSS and 2*WSS have passed",
			args: args{
				epoch:        wssPeriod,
				highestEpoch: 2 * wssPeriod,
			},
			want: true,
		},
		{
			name: "returns true if epoch == 2*WSS and 3*WSS have passed",
			args: args{
				epoch:        2 * wssPeriod,
				highestEpoch: 3 * wssPeriod,
			},
			want: true,
		},
		{
			name: "returns false if epoch == 3*WSS and 3*WSS have passed",
			args: args{
				epoch:        3 * wssPeriod,
				highestEpoch: 3 * wssPeriod,
			},
			want: false,
		},
		{
			name: "returns false if epoch == 4*WSS and 3*WSS have passed",
			args: args{
				epoch:        4 * wssPeriod,
				highestEpoch: 3 * wssPeriod,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := olderThanCurrentWeakSubjectivityPeriod(tt.args.epoch, tt.args.highestEpoch); got != tt.want {
				t.Errorf("olderThanCurrentWeakSubjectivityPeriod() = %v, want %v", got, tt.want)
			}
		})
	}
}
