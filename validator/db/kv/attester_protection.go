package kv

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/slashutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SlashingKind used for helpful information upon detection.
type SlashingKind int

// An attestation record can be represented by these simple values
// for manipulation by database methods.
type attestationRecord struct {
	pubKey      [48]byte
	source      uint64
	target      uint64
	signingRoot [32]byte
}

// A wrapper over an error received from a background routine
// saving batched attestations for slashing protection.
// This wrapper allows us to send this response over event feeds,
// as our event feed does not allow sending `nil` values to
// subscribers.
type saveAttestationsResponse struct {
	err error
}

// Enums representing the types of slashable events for attesters.
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
	ctx context.Context, pubKey [48]byte, signingRoot [32]byte, att *ethpb.IndexedAttestation,
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
			targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(att.Data.Target.Epoch)
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
		return sourceEpochsBucket.ForEach(func(sourceEpochBytes []byte, targetEpochsBytes []byte) error {
			existingSourceEpoch := bytesutil.BytesToUint64BigEndian(sourceEpochBytes)

			// There can be multiple target epochs attested per source epoch.
			attestedTargetEpochs := make([]uint64, 0)
			for i := 0; i < len(targetEpochsBytes); i += 8 {
				targetEpoch := bytesutil.BytesToUint64BigEndian(targetEpochsBytes[i : i+8])
				attestedTargetEpochs = append(attestedTargetEpochs, targetEpoch)
			}

			for _, existingTargetEpoch := range attestedTargetEpochs {
				existingAtt := &ethpb.IndexedAttestation{
					Data: &ethpb.AttestationData{
						Source: &ethpb.Checkpoint{Epoch: existingSourceEpoch},
						Target: &ethpb.Checkpoint{Epoch: existingTargetEpoch},
					},
				}
				// Checks if the incoming attestation is surrounding or
				// is surrounded by an existing one.
				surrounding := slashutil.IsSurround(att, existingAtt)
				surrounded := slashutil.IsSurround(existingAtt, att)
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
			}
			return nil
		})
	})
	return slashKind, err
}

// SaveAttestationForPubKey saves an attestation for a validator public
// key for local validator slashing protection.
func (store *Store) SaveAttestationForPubKey(
	ctx context.Context, pubKey [48]byte, signingRoot [32]byte, att *ethpb.IndexedAttestation,
) error {
	store.batchedAttestationsChan <- &attestationRecord{
		pubKey:      pubKey,
		source:      att.Data.Source.Epoch,
		target:      att.Data.Target.Epoch,
		signingRoot: signingRoot,
	}
	// Subscribe to be notified when the attestation record queued
	// for saving to the DB is indeed saved. If an error occurred
	// during the process of saving the attestation record, the sender
	// will give us that error. We use a buffered channel
	// to prevent blocking the sender from notifying us of the result.
	responseChan := make(chan saveAttestationsResponse, 1)
	defer close(responseChan)
	sub := store.batchAttestationsFlushedFeed.Subscribe(responseChan)
	defer sub.Unsubscribe()
	res := <-responseChan
	return res.err
}

// Meant to run as a background routine, this function checks whether:
// (a) we have reached a max capacity of batched attestations in the Store or
// (b) attestationBatchWriteInterval has passed
// Based on whichever comes first, this function then proceeds
// to flush the attestations to the DB all at once in a single boltDB
// transaction for efficiency. Then, batched attestations slice is emptied out.
func (store *Store) batchAttestationWrites(ctx context.Context) {
	ticker := time.NewTicker(attestationBatchWriteInterval)
	defer ticker.Stop()
	for {
		select {
		case v := <-store.batchedAttestationsChan:
			store.batchedAttestations = append(store.batchedAttestations, v)
			if len(store.batchedAttestations) == attestationBatchCapacity {
				log.WithField("numRecords", attestationBatchCapacity).Debug(
					"Reached max capacity of batched attestation records, flushing to DB",
				)
				store.flushAttestationRecords(ctx)
			}
		case <-ticker.C:
			if len(store.batchedAttestations) > 0 {
				log.WithField("numRecords", len(store.batchedAttestations)).Debug(
					"Batched attestation records write interval reached, flushing to DB",
				)
				store.flushAttestationRecords(ctx)
			}
		case <-ctx.Done():
			return
		}
	}
}

// Flushes a list of batched attestations to the database
// and resets the list of batched attestations for future writes.
// This function notifies all subscribers for flushed attestations
// of the result of the save operation.
func (store *Store) flushAttestationRecords(ctx context.Context) {
	err := store.saveAttestationRecords(ctx, store.batchedAttestations)
	// If there was no error, we reset the batched attestations slice.
	if err == nil {
		log.Debug("Successfully flushed batched attestations to DB")
		store.batchedAttestations = make([]*attestationRecord, 0, attestationBatchCapacity)
	}
	// Forward the error, if any, to all subscribers via an event feed.
	// We use a struct wrapper around the error as the event feed
	// cannot handle sending a raw `nil` in case there is no error.
	store.batchAttestationsFlushedFeed.Send(saveAttestationsResponse{
		err: err,
	})
}

// Saves a list of attestation records to the database in a single boltDB
// transaction to minimize write lock contention compared to doing them
// all in individual, isolated boltDB transactions.
func (store *Store) saveAttestationRecords(ctx context.Context, atts []*attestationRecord) error {
	ctx, span := trace.StartSpan(ctx, "Validator.saveAttestationRecords")
	defer span.End()
	return store.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		for _, att := range atts {
			pkBucket, err := bucket.CreateBucketIfNotExists(att.pubKey[:])
			if err != nil {
				return errors.Wrap(err, "could not create public key bucket")
			}
			sourceEpochBytes := bytesutil.Uint64ToBytesBigEndian(att.source)
			targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(att.target)

			signingRootsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
			if err != nil {
				return errors.Wrap(err, "could not create signing roots bucket")
			}
			if err := signingRootsBucket.Put(targetEpochBytes, att.signingRoot[:]); err != nil {
				return errors.Wrapf(err, "could not save signing signing root for epoch %d", att.target)
			}
			sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
			if err != nil {
				return errors.Wrap(err, "could not create source epochs bucket")
			}

			// There can be multiple attested target epochs per source epoch.
			// If a previous list exists, we append to that list with the incoming target epoch.
			// Otherwise, we initialize it using the incoming target epoch.
			var existingAttestedTargetsBytes []byte
			if existing := sourceEpochsBucket.Get(sourceEpochBytes); existing != nil {
				existingAttestedTargetsBytes = append(existing, targetEpochBytes...)
			} else {
				existingAttestedTargetsBytes = targetEpochBytes
			}

			if err := sourceEpochsBucket.Put(sourceEpochBytes, existingAttestedTargetsBytes); err != nil {
				return errors.Wrapf(err, "could not save source epoch %d for epoch %d", att.source, att.target)
			}
			// Initialize buckets for the lowest target and source epochs.
			lowestSourceBucket, err := tx.CreateBucketIfNotExists(lowestSignedSourceBucket)
			if err != nil {
				return err
			}
			lowestTargetBucket, err := tx.CreateBucketIfNotExists(lowestSignedTargetBucket)
			if err != nil {
				return err
			}

			// If the incoming source epoch is lower than the lowest signed source epoch, override.
			lowestSignedSourceBytes := lowestSourceBucket.Get(att.pubKey[:])
			var lowestSignedSourceEpoch uint64
			if len(lowestSignedSourceBytes) >= 8 {
				lowestSignedSourceEpoch = bytesutil.BytesToUint64BigEndian(lowestSignedSourceBytes)
			}
			if len(lowestSignedSourceBytes) == 0 || att.source < lowestSignedSourceEpoch {
				if err := lowestSourceBucket.Put(
					att.pubKey[:], bytesutil.Uint64ToBytesBigEndian(att.source),
				); err != nil {
					return err
				}
			}

			// If the incoming target epoch is lower than the lowest signed target epoch, override.
			lowestSignedTargetBytes := lowestTargetBucket.Get(att.pubKey[:])
			var lowestSignedTargetEpoch uint64
			if len(lowestSignedTargetBytes) >= 8 {
				lowestSignedTargetEpoch = bytesutil.BytesToUint64BigEndian(lowestSignedTargetBytes)
			}
			if len(lowestSignedTargetBytes) == 0 || att.target < lowestSignedTargetEpoch {
				if err := lowestTargetBucket.Put(
					att.pubKey[:], bytesutil.Uint64ToBytesBigEndian(att.target),
				); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// AttestedPublicKeys retrieves all public keys in our attestation history bucket.
func (store *Store) AttestedPublicKeys(ctx context.Context) ([][48]byte, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.AttestedPublicKeys")
	defer span.End()
	var err error
	attestedPublicKeys := make([][48]byte, 0)
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(deprecatedAttestationHistoryBucket)
		return bucket.ForEach(func(key []byte, _ []byte) error {
			pubKeyBytes := [48]byte{}
			copy(pubKeyBytes[:], key)
			attestedPublicKeys = append(attestedPublicKeys, pubKeyBytes)
			return nil
		})
	})
	return attestedPublicKeys, err
}

// LowestSignedSourceEpoch returns the lowest signed source epoch for a validator public key.
// If no data exists, returning 0 is a sensible default.
func (store *Store) LowestSignedSourceEpoch(ctx context.Context, publicKey [48]byte) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.LowestSignedSourceEpoch")
	defer span.End()

	var err error
	var lowestSignedSourceEpoch uint64
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(lowestSignedSourceBucket)
		lowestSignedSourceBytes := bucket.Get(publicKey[:])
		// 8 because bytesutil.BytesToUint64BigEndian will return 0 if input is less than 8 bytes.
		if len(lowestSignedSourceBytes) < 8 {
			return nil
		}
		lowestSignedSourceEpoch = bytesutil.BytesToUint64BigEndian(lowestSignedSourceBytes)
		return nil
	})
	return lowestSignedSourceEpoch, err
}

// LowestSignedTargetEpoch returns the lowest signed target epoch for a validator public key.
// If no data exists, returning 0 is a sensible default.
func (store *Store) LowestSignedTargetEpoch(ctx context.Context, publicKey [48]byte) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.LowestSignedTargetEpoch")
	defer span.End()

	var err error
	var lowestSignedTargetEpoch uint64
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(lowestSignedTargetBucket)
		lowestSignedTargetBytes := bucket.Get(publicKey[:])
		// 8 because bytesutil.BytesToUint64BigEndian will return 0 if input is less than 8 bytes.
		if len(lowestSignedTargetBytes) < 8 {
			return nil
		}
		lowestSignedTargetEpoch = bytesutil.BytesToUint64BigEndian(lowestSignedTargetBytes)
		return nil
	})
	return lowestSignedTargetEpoch, err
}
