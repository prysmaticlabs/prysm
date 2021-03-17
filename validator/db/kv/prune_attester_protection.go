package kv

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// PruneAttestations loops through every public key in the public keys bucket
// and prunes all attestation data that has target epochs older the highest
// target epoch minus some constant of how many epochs we keep track of for slashing
// protection. This routine is meant to run on startup.
func (s *Store) PruneAttestations(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "Validator.PruneAttestations")
	defer span.End()
	pubkeys := [][]byte{}
	err := s.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		return bucket.ForEach(func(pubKey []byte, _ []byte) error {
			key := make([]byte, len(pubKey))
			copy(key, pubKey)
			pubkeys = append(pubkeys, pubKey)
			return nil
		})
	})
	if err != nil {
		return err
	}
	for _, k := range pubkeys {
		err = s.update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(pubKeysBucket)
			pkBucket := bucket.Bucket(k)
			if pkBucket == nil {
				return nil
			}
			if err := pruneSourceEpochsBucket(pkBucket); err != nil {
				return err
			}
			if err := pruneTargetEpochsBucket(pkBucket); err != nil {
				return err
			}
			return pruneSigningRootsBucket(pkBucket)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func pruneSourceEpochsBucket(bucket *bolt.Bucket) error {
	sourceEpochsBucket := bucket.Bucket(attestationSourceEpochsBucket)
	if sourceEpochsBucket == nil {
		return nil
	}
	// We obtain the highest source epoch from the source epochs bucket.
	// Then, we obtain the corresponding target epoch for that source epoch.
	highestSourceEpochBytes, _ := sourceEpochsBucket.Cursor().Last()
	highestTargetEpochBytes := sourceEpochsBucket.Get(highestSourceEpochBytes)
	highestTargetEpoch := bytesutil.BytesToEpochBigEndian(highestTargetEpochBytes)

	return sourceEpochsBucket.ForEach(func(k []byte, v []byte) error {
		targetEpoch := bytesutil.BytesToEpochBigEndian(v)
		if targetEpoch < pruningEpochCutoff(highestTargetEpoch) {
			return sourceEpochsBucket.Delete(k)
		}
		return nil
	})
}

func pruneTargetEpochsBucket(bucket *bolt.Bucket) error {
	targetEpochsBucket := bucket.Bucket(attestationTargetEpochsBucket)
	if targetEpochsBucket == nil {
		return nil
	}
	// We obtain the highest target epoch from the bucket.
	highestTargetEpochBytes, _ := targetEpochsBucket.Cursor().Last()
	highestTargetEpoch := bytesutil.BytesToEpochBigEndian(highestTargetEpochBytes)

	return targetEpochsBucket.ForEach(func(k []byte, v []byte) error {
		targetEpoch := bytesutil.BytesToEpochBigEndian(k)
		if targetEpoch < pruningEpochCutoff(highestTargetEpoch) {
			return targetEpochsBucket.Delete(k)
		}
		return nil
	})
}

func pruneSigningRootsBucket(bucket *bolt.Bucket) error {
	signingRootsBucket := bucket.Bucket(attestationSigningRootsBucket)
	if signingRootsBucket == nil {
		return nil
	}

	// We obtain the highest target epoch from the signing roots bucket.
	highestTargetEpochBytes, _ := signingRootsBucket.Cursor().Last()
	highestTargetEpoch := bytesutil.BytesToEpochBigEndian(highestTargetEpochBytes)

	return signingRootsBucket.ForEach(func(k []byte, v []byte) error {
		targetEpoch := bytesutil.BytesToEpochBigEndian(k)
		if targetEpoch < pruningEpochCutoff(highestTargetEpoch) {
			return signingRootsBucket.Delete(k)
		}
		return nil
	})
}

// This helper function determines the cutoff epoch where, for all epochs before it, we should prune
// the slashing protection database. This is computed by taking in an epoch and subtracting
// SLASHING_PROTECTION_PRUNING_EPOCHS from the value. For example, if we are keeping track of 512 epochs
// in the database, if we pass in epoch 612, then we want to prune all epochs before epoch 100.
func pruningEpochCutoff(epoch types.Epoch) types.Epoch {
	minEpoch := types.Epoch(0)
	if epoch > params.BeaconConfig().SlashingProtectionPruningEpochs {
		minEpoch = epoch - params.BeaconConfig().SlashingProtectionPruningEpochs
	}
	return minEpoch
}
