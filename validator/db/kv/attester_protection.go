package kv

import (
	"bytes"
	"context"
	"fmt"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/slashutil"
	bolt "go.etcd.io/bbolt"
)

// SlashingKind used for helpful information upon detection.
type SlashingKind int

type attestationRecord struct {
	pubKey      [48]byte
	source      uint64
	target      uint64
	signingRoot [32]byte
}

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
		return sourceEpochsBucket.ForEach(func(sourceEpochBytes []byte, targetEpochBytes []byte) error {
			existingSourceEpoch := bytesutil.BytesToUint64BigEndian(sourceEpochBytes)
			existingTargetEpoch := bytesutil.BytesToUint64BigEndian(targetEpochBytes)
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
			return nil
		})
	})
	return slashKind, err
}

// ApplyAttestationForPubKey applies an attestation for a validator public
// key by storing its signing root under the appropriate bucket as well
// as its source and target epochs for future slashing protection checks.
func (store *Store) ApplyAttestationForPubKey(
	ctx context.Context, pubKey [48]byte, signingRoot [32]byte, att *ethpb.IndexedAttestation,
) error {
	return nil
}

func (store *Store) batchAttestationWrites() error {
	ch := make(chan *attestationRecord, 100)
	listeners := make([]chan struct{}, 0)
	flushDelay := time.Millisecond * 100
	timer := time.NewTimer(flushDelay)
	defer timer.Stop()
	attestations := make([]*attestationRecord, 0, 100)
	for {
		select {
		case v := <-ch:
			fmt.Println("Received attestation")
			attestations = append(attestations, v)
			if len(attestations) == cap(attestations) {
				fmt.Println("Atts slice is full, need to flush")
				if err := store.flushAttestationsToDB(attestations); err != nil {
					_ = err
				}
				for _, l := range listeners {
					l <- struct{}{}
				}
				timer.Reset(flushDelay)
			}
		case <-timer.C:
			if len(attestations) > 0 {
				fmt.Println("Delay passed, forcing flush")
				if err := store.flushAttestationsToDB(attestations); err != nil {
					_ = err
				}
				for _, l := range listeners {
					l <- struct{}{}
				}
			}
			timer.Reset(flushDelay)
		}
	}
}

func (store *Store) flushAttestationsToDB(atts []*attestationRecord) error {
	tx, err := store.db.Begin(true /* writable */)
	if err != nil {
		return err
	}
	bucket := tx.Bucket(pubKeysBucket)
	for _, att := range atts {
		pkBucket, err := bucket.CreateBucketIfNotExists(att.pubKey[:])
		if err != nil {
			return err
		}
		sourceEpochBytes := bytesutil.Uint64ToBytesBigEndian(att.source)
		targetEpochBytes := bytesutil.Uint64ToBytesBigEndian(att.target)

		signingRootsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSigningRootsBucket)
		if err != nil {
			return err
		}
		if err := signingRootsBucket.Put(targetEpochBytes, att.signingRoot[:]); err != nil {
			return err
		}
		sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
		if err != nil {
			return err
		}
		if err := sourceEpochsBucket.Put(sourceEpochBytes, targetEpochBytes); err != nil {
			return err
		}
	}
	return tx.Commit()
}
