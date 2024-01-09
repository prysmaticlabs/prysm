package kv

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

var (
	errBlobSlotMismatch     = errors.New("sidecar slot mismatch")
	errBlobParentMismatch   = errors.New("sidecar parent root mismatch")
	errBlobRootMismatch     = errors.New("sidecar root mismatch")
	errBlobProposerMismatch = errors.New("sidecar proposer index mismatch")
	errBlobSidecarLimit     = errors.New("sidecar exceeds maximum number of blobs")
	errEmptySidecar         = errors.New("nil or empty blob sidecars")
	errNewerBlobExists      = errors.New("Will not overwrite newer blobs in db")
)

// A blob rotating key is represented as bytes(slot_to_rotating_buffer(blob.slot)) ++ bytes(blob.slot) ++ blob.block_root
type blobRotatingKey []byte

// BufferPrefix returns the first 8 bytes of the rotating key.
// This represents bytes(slot_to_rotating_buffer(blob.slot)) in the rotating key.
func (rk blobRotatingKey) BufferPrefix() []byte {
	return rk[0:8]
}

// Slot returns the information from the key.
func (rk blobRotatingKey) Slot() types.Slot {
	slotBytes := rk[8:16]
	return bytesutil.BytesToSlotBigEndian(slotBytes)
}

// BlockRoot returns the block root information from the key.
func (rk blobRotatingKey) BlockRoot() []byte {
	return rk[16:]
}

// DeleteBlobSidecars returns true if the blobs are in the db.
func (s *Store) DeleteBlobSidecars(ctx context.Context, beaconBlockRoot [32]byte) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.DeleteBlobSidecar")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blobsBucket)
		c := bkt.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if bytes.HasSuffix(k, beaconBlockRoot[:]) {
				if err := bkt.Delete(k); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *Store) slotKey(slot types.Slot) []byte {
	return bytesutil.SlotToBytesBigEndian(slot.ModSlot(s.blobRetentionSlots()))
}

func (s *Store) blobRetentionSlots() types.Slot {
	return types.Slot(s.blobRetentionEpochs.Mul(uint64(params.BeaconConfig().SlotsPerEpoch)))
}

var errBlobRetentionEpochMismatch = errors.New("epochs for blobs request value in DB does not match runtime config")

func (s *Store) checkEpochsForBlobSidecarsRequestBucket(db *bolt.DB) error {
	uRetentionEpochs := uint64(s.blobRetentionEpochs)
	if err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(chainMetadataBucket)
		v := b.Get(blobRetentionEpochsKey)
		if v == nil {
			if err := b.Put(blobRetentionEpochsKey, bytesutil.Uint64ToBytesBigEndian(uRetentionEpochs)); err != nil {
				return err
			}
			return nil
		}
		e := bytesutil.BytesToUint64BigEndian(v)
		if e != uRetentionEpochs {
			return errors.Wrapf(errBlobRetentionEpochMismatch, "db=%d, config=%d", e, uRetentionEpochs)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
