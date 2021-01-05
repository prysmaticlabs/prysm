package kv

import (
	"context"
	"fmt"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	bolt "go.etcd.io/bbolt"
)

func TestStore_CheckSurroundVote_54kEpochs(t *testing.T) {
	ctx := context.Background()
	numValidators := 1
	numEpochs := uint64(54000)
	pubKeys := make([][48]byte, numValidators)
	validatorDB := setupDB(t, pubKeys)

	// Attest to every (source = epoch, target = epoch + 1) sequential pair
	// since genesis up to and including the weak subjectivity period epoch (54,000).
	err := validatorDB.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		sourceEpochsBucket, err := bucket.CreateBucketIfNotExists(pubKeys[0][:])
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

	surroundingAtt := createAttestation(numEpochs/2, numEpochs /* source, target */)
	start := time.Now()
	slashable := validatorDB.CheckSlashableAttestation(ctx, pubKeys[0], numEpochs/2, numEpochs)
	assert.Equal(t, true, slashable)
	end := time.Now()
	fmt.Printf("Checking surround vote with (source %d, target %d), took %v\n", surroundingAtt.Data.Source.Epoch, surroundingAtt.Data.Target.Epoch, end.Sub(start))

	surroundingAtt = createAttestation(0, numEpochs /* source, target */)
	start = time.Now()
	slashable = validatorDB.CheckSurroundVote(ctx, pubKeys[0], surroundingAtt)
	assert.Equal(t, true, slashable)
	end = time.Now()
	fmt.Printf("Checking surround vote with (source %d, target %d), took %v\n", surroundingAtt.Data.Source.Epoch, surroundingAtt.Data.Target.Epoch, end.Sub(start))

	surroundingAtt = createAttestation(numEpochs-3, numEpochs /* source, target */)
	start = time.Now()
	slashable = validatorDB.CheckSurroundVote(ctx, pubKeys[0], surroundingAtt)
	assert.Equal(t, true, slashable)
	end = time.Now()
	fmt.Printf("Checking surround vote with (source %d, target %d), took %v\n", surroundingAtt.Data.Source.Epoch, surroundingAtt.Data.Target.Epoch, end.Sub(start))

	safeAtt := createAttestation(numEpochs, numEpochs+1 /* source, target */)
	start = time.Now()
	slashable = validatorDB.CheckSurroundVote(ctx, pubKeys[0], safeAtt)
	assert.Equal(t, false, slashable)
	end = time.Now()
	fmt.Printf("Checking safe attestation with (source %d, target %d), took %v\n", safeAtt.Data.Source.Epoch, safeAtt.Data.Target.Epoch, end.Sub(start))
}

func BenchmarkStore_CheckSurroundVote_SafeAttestation_54kEpochs(b *testing.B) {
	numValidators := 1
	numEpochs := uint64(54000)
	pubKeys := make([][48]byte, numValidators)
	benchCheckSurroundVote(b, pubKeys, numEpochs, false /* surround */)
}

func BenchmarkStore_CheckSurroundVote_Surrounding_54kEpochs(b *testing.B) {
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
	validatorDB, err := NewKVStore(ctx, "/tmp/benchbench", pubKeys)
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
			sourceEpochsBucket, err := bucket.CreateBucketIfNotExists(pubKey[:])
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
			slashable := validatorDB.CheckSurroundVote(ctx, pubKey, surroundingVote)
			assert.Equal(b, shouldSurround, slashable)
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
