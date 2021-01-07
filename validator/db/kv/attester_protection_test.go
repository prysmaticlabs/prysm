package kv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	bolt "go.etcd.io/bbolt"
)

func TestStore_CheckSlashableAttestation_DoubleVote(t *testing.T) {
	ctx := context.Background()
	numValidators := 1
	pubKeys := make([][48]byte, numValidators)
	validatorDB := setupDB(t, pubKeys)
	tests := []struct {
		name                string
		existingAttestation *ethpb.Attestation
		existingSigningRoot [32]byte
		incomingAttestation *ethpb.Attestation
		incomingSigningRoot [32]byte
		want                bool
	}{
		{
			name:                "different signing root at same target equals a double vote",
			existingAttestation: createAttestation(0, 1 /* target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestation(0, 1 /* target */),
			incomingSigningRoot: [32]byte{2},
			want:                true,
		},
		{
			name:                "same signing root at same target is safe",
			existingAttestation: createAttestation(0, 1 /* target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestation(0, 1 /* target */),
			incomingSigningRoot: [32]byte{1},
			want:                false,
		},
		{
			name:                "different signing root at different target is safe",
			existingAttestation: createAttestation(0, 1 /* target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestation(0, 2 /* target */),
			incomingSigningRoot: [32]byte{2},
			want:                false,
		},
		{
			name:                "no data stored at target should not be considered a double vote",
			existingAttestation: createAttestation(0, 1 /* target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestation(0, 2 /* target */),
			incomingSigningRoot: [32]byte{1},
			want:                false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatorDB.ApplyAttestationForPubKey(
				ctx,
				pubKeys[0],
				tt.existingSigningRoot,
				tt.existingAttestation,
			)
			require.NoError(t, err)
			slashingKind, err := validatorDB.CheckSlashableAttestation(
				ctx,
				pubKeys[0],
				tt.incomingSigningRoot,
				tt.incomingAttestation,
			)
			if tt.want {
				require.NotNil(t, err)
				assert.Equal(t, DoubleVote, slashingKind)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStore_CheckSlashableAttestation_SurroundVote_54kEpochs(t *testing.T) {
	ctx := context.Background()
	numValidators := 1
	numEpochs := uint64(54000)
	pubKeys := make([][48]byte, numValidators)
	validatorDB := setupDB(t, pubKeys)

	// Attest to every (source = epoch, target = epoch + 1) sequential pair
	// since genesis up to and including the weak subjectivity period epoch (54,000).
	err := validatorDB.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBucket, err := bucket.CreateBucketIfNotExists(pubKeys[0][:])
		if err != nil {
			return err
		}
		sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
		if err != nil {
			return err
		}
		for epoch := uint64(1); epoch < numEpochs; epoch++ {
			att := createAttestation(epoch-1, epoch)
			sourceEpoch := bytesutil.Uint64ToBytesBigEndian(att.Data.Source.Epoch)
			targetEpoch := bytesutil.Uint64ToBytesBigEndian(att.Data.Target.Epoch)
			if err := sourceEpochsBucket.Put(sourceEpoch, targetEpoch); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		signingRoot [32]byte
		attestation *ethpb.Attestation
		want        SlashingKind
	}{
		{
			name:        "surround vote at half of the weak subjectivity period",
			signingRoot: [32]byte{},
			attestation: createAttestation(numEpochs/2, numEpochs),
			want:        SurroundingVote,
		},
		{
			name:        "spanning genesis to weak subjectivity period surround vote",
			signingRoot: [32]byte{},
			attestation: createAttestation(0, numEpochs),
			want:        SurroundingVote,
		},
		{
			name:        "simple surround vote at end of weak subjectivity period",
			signingRoot: [32]byte{},
			attestation: createAttestation(numEpochs-3, numEpochs),
			want:        SurroundingVote,
		},
		{
			name:        "non-slashable vote",
			signingRoot: [32]byte{},
			attestation: createAttestation(numEpochs, numEpochs+1),
			want:        NotSlashable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slashingKind, err := validatorDB.CheckSlashableAttestation(ctx, pubKeys[0], tt.signingRoot, tt.attestation)
			if tt.want != NotSlashable {
				require.NotNil(t, err)
			}
			assert.Equal(t, tt.want, slashingKind)
		})
	}
}

func TestPruneAttestationsOlderThanCurrentWeakSubjectivity(t *testing.T) {
	ctx := context.Background()
	numValidators := 1
	numEpochs := params.BeaconConfig().WeakSubjectivityPeriod
	pubKeys := make([][48]byte, numValidators)
	validatorDB := setupDB(t, pubKeys)

	// Write signing roots for every single epoch
	// since genesis to WEAK_SUBJECTIVITY_PERIOD.
	err := validatorDB.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBucket, err := bucket.CreateBucketIfNotExists(pubKeys[0][:])
		if err != nil {
			return err
		}
		signingRootsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
		if err != nil {
			return err
		}
		for epoch := uint64(0); epoch < numEpochs; epoch++ {
			targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(epoch)
			var signingRoot [32]byte
			copy(signingRoot[:], fmt.Sprintf("%d", targetEpochBytes))
			if err := signingRootsBucket.Put(targetEpochBytes, signingRoot[:]); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	// Next, attempt to prune and realize that we still have all epochs intact.
}

func BenchmarkStore_CheckSlashableAttestation_Surround_SafeAttestation_54kEpochs(b *testing.B) {
	numValidators := 1
	numEpochs := uint64(54000)
	pubKeys := make([][48]byte, numValidators)
	benchCheckSurroundVote(b, pubKeys, numEpochs, false /* surround */)
}

func BenchmarkStore_CheckSurroundVote_Surround_Slashable_54kEpochs(b *testing.B) {
	numValidators := 1
	numEpochs := uint64(54000)
	pubKeys := make([][48]byte, numValidators)
	benchCheckSurroundVote(b, pubKeys, numEpochs, true /* surround */)
}

func benchCheckSurroundVote(
	b *testing.B,
	pubKeys [][48]byte,
	numEpochs uint64,
	shouldSurround bool,
) {
	ctx := context.Background()
	validatorDB, err := NewKVStore(ctx, filepath.Join(os.TempDir(), "benchsurroundvote"), pubKeys)
	require.NoError(b, err, "Failed to instantiate DB")
	defer func() {
		require.NoError(b, validatorDB.Close(), "Failed to close database")
		require.NoError(b, validatorDB.ClearDB(), "Failed to clear database")
	}()
	// Every validator will have attested every (source, target) sequential pair
	// since genesis up to and including the weak subjectivity period epoch (54,000).
	err = validatorDB.update(func(tx *bolt.Tx) error {
		for _, pubKey := range pubKeys {
			bucket := tx.Bucket(pubKeysBucket)
			pkBucket, err := bucket.CreateBucketIfNotExists(pubKey[:])
			if err != nil {
				return err
			}
			sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
			if err != nil {
				return err
			}
			for epoch := uint64(1); epoch < numEpochs; epoch++ {
				att := createAttestation(epoch-1, epoch)
				sourceEpoch := bytesutil.Uint64ToBytesBigEndian(att.Data.Source.Epoch)
				targetEpoch := bytesutil.Uint64ToBytesBigEndian(att.Data.Target.Epoch)
				if err := sourceEpochsBucket.Put(sourceEpoch, targetEpoch); err != nil {
					return err
				}
			}
		}
		return nil
	})
	require.NoError(b, err)

	// Will surround many attestations.
	var surroundingVote *ethpb.Attestation
	if shouldSurround {
		surroundingVote = createAttestation(numEpochs/2, numEpochs)
	} else {
		surroundingVote = createAttestation(numEpochs+1, numEpochs+2)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pubKey := range pubKeys {
			slashingKind, err := validatorDB.CheckSlashableAttestation(ctx, pubKey, [32]byte{}, surroundingVote)
			if shouldSurround {
				require.NotNil(b, err)
				assert.Equal(b, SurroundingVote, slashingKind)
			} else {
				require.NoError(b, err)
			}
		}
	}
}

func createAttestation(source, target uint64) *ethpb.Attestation {
	return &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{
				Epoch: source,
			},
			Target: &ethpb.Checkpoint{
				Epoch: target,
			},
		},
	}
}
