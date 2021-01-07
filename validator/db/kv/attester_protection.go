package kv

import (
	"bytes"
	"context"
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
)

// SlashingKind used for helpful information upon detection.
type SlashingKind int

const (
	NotSlashable SlashingKind = iota
	DoubleVote
	SurroundingVote
	SurroundedVote
)

var (
	doubleVoteMessage      = "double vote found, existing attestation at target epoch %d with conflicting signing root %#x"
	surroundingVoteMessage = "attestation with (source %d, target %d) surrounds another with (source %d, target %d)"
	surroundedVoteMessage  = "attestation with (source %d, target %d) is surrounded by another with (source %d, target %d)"
)

// CheckSlashableAttestation verifies an incoming attestation is
// not a double vote for a validator public key nor a surround vote.
func (store *Store) CheckSlashableAttestation(
	ctx context.Context, pubKey [48]byte, signingRoot [32]byte, att *ethpb.Attestation,
) (SlashingKind, error) {
	var slashKind SlashingKind
	err := store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBucket := bucket.Bucket(pubKey[:])
		if pkBucket == nil {
			return nil
		}

		// First we check for double votes.
		signingRootsBucket := pkBucket.Bucket(attestationSigningRootsBucket)
		if signingRootsBucket != nil {
			targetEpoch := att.Data.Target.Epoch % params.BeaconConfig().WeakSubjectivityPeriod
			targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(targetEpoch)
			existingSigningRoot := signingRootsBucket.Get(targetEpochBytes)
			if existingSigningRoot != nil && !bytes.Equal(signingRoot[:], existingSigningRoot) {
				slashKind = DoubleVote
				return fmt.Errorf(doubleVoteMessage, att.Data.Target.Epoch, existingSigningRoot)
			}
		}

		sourceEpochsBucket := pkBucket.Bucket(attestationSourceEpochsBucket)
		if sourceEpochsBucket == nil {
			return nil
		}
		// Check for surround votes.
		return sourceEpochsBucket.ForEach(func(sourceEpochBytes []byte, targetEpochBytes []byte) error {
			existingSourceEpoch := bytesutil.BytesToUint64BigEndian(sourceEpochBytes)
			existingTargetEpoch := bytesutil.BytesToUint64BigEndian(targetEpochBytes)
			surrounding := att.Data.Source.Epoch < existingSourceEpoch && att.Data.Target.Epoch > existingTargetEpoch
			surrounded := att.Data.Source.Epoch > existingSourceEpoch && att.Data.Target.Epoch < existingTargetEpoch
			if surrounding {
				slashKind = SurroundingVote
				return fmt.Errorf(
					surroundingVoteMessage,
					att.Data.Source.Epoch,
					att.Data.Target.Epoch,
					existingSourceEpoch,
					existingTargetEpoch,
				)
			}
			if surrounded {
				slashKind = SurroundedVote
				return fmt.Errorf(
					surroundedVoteMessage,
					att.Data.Source.Epoch,
					att.Data.Target.Epoch,
					existingSourceEpoch,
					existingTargetEpoch,
				)
			}
			return nil
		})
	})
	return slashKind, err
}

// ApplyAttestationForPubKey applies an attestation for a validator public
// key by storing its signing root under the appropriate bucket as well
// as its source and target epochs for future slashing protection checks.
func (store *Store) ApplyAttestationForPubKey(
	ctx context.Context, pubKey [48]byte, signingRoot [32]byte, att *ethpb.Attestation,
) error {
	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBucket, err := bucket.CreateBucketIfNotExists(pubKey[:])
		if err != nil {
			return err
		}
		sourceEpochBytes := bytesutil.Uint64ToBytesBigEndian(att.Data.Source.Epoch)
		targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(att.Data.Target.Epoch)

		signingRootsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
		if err != nil {
			return err
		}
		if err := signingRootsBucket.Put(targetEpochBytes, signingRoot[:]); err != nil {
			return err
		}
		sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
		if err != nil {
			return err
		}
		return sourceEpochsBucket.Put(sourceEpochBytes, targetEpochBytes)
	})
}

// PruneAttestationsOlderThanCurrentWeakSubjectivity loops through every
// public key in the public keys bucket and prunes all attestation signing roots
// that are older than the highest written epoch % weak subjectivity period.
// This routine is meant to run on startup.
func (store *Store) PruneAttestationsOlderThanCurrentWeakSubjectivity(ctx context.Context) error {
	wssPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		// Loop through every immediate subbucket, which corresponds to
		// each public key in the database.
		return bucket.ForEach(func(pubKey []byte, _ []byte) error {
			pkBucket := bucket.Bucket(pubKey)
			if pkBucket == nil {
				return nil
			}
			signingRootsBucket := pkBucket.Bucket(attestationSigningRootsBucket)
			sourceEpochsBucket := pkBucket.Bucket(attestationSourceEpochsBucket)

			// We obtain the highest target epoch from the signing roots bucket.
			highestTargetSigningRootEpochBytes, _ := signingRootsBucket.Cursor().Last()
			highestTargetSigningRootEpoch := bytesutil.BytesToUint64BigEndian(highestTargetSigningRootEpochBytes)
			numWssPeriodsForSigningRootTargetEpoch := highestTargetSigningRootEpoch / wssPeriod

			// We obtain the highest source epoch from the signing roots bucket.
			highestTargetEpochBytes, _ := sourceEpochsBucket.Cursor().Last()
			highestTargetEpoch := bytesutil.BytesToUint64BigEndian(highestTargetEpochBytes)
			numWssPeriodsForTargetEpoch := highestTargetEpoch / wssPeriod

			// If the highest target epoch is greater than WEAK_SUBJECTIVITY_PERIOD,
			// this means we can start pruning old attestation signing roots.
			if highestTargetEpoch > wssPeriod {
				if err := sourceEpochsBucket.ForEach(func(k []byte, v []byte) error {
					targetEpoch := bytesutil.BytesToUint64BigEndian(v)

					// For each attestation signing root we find, we check
					// if it less than the weak subjectivity period of the
					// highest written target epoch in the bucket and delete if so.
					if targetEpoch < wssPeriod {
						return sourceEpochsBucket.Delete(k)
					} else if (targetEpoch / wssPeriod) < numWssPeriodsForTargetEpoch {
						return sourceEpochsBucket.Delete(k)
					}
					return nil
				}); err != nil {
					return err
				}
			}
			if highestTargetSigningRootEpoch > wssPeriod {
				return signingRootsBucket.ForEach(func(k []byte, _ []byte) error {
					targetEpoch := bytesutil.BytesToUint64BigEndian(k)

					// For each attestation signing root we find, we check
					// if it less than the weak subjectivity period of the
					// highest written target epoch in the bucket and delete if so.
					if targetEpoch < wssPeriod {
						return signingRootsBucket.Delete(k)
					} else if (targetEpoch / wssPeriod) < numWssPeriodsForSigningRootTargetEpoch {
						return signingRootsBucket.Delete(k)
					}
					return nil
				})
			}
			return nil
		})
	})
}
