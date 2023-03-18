package kv

import (
	"bytes"
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/config/params"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

const blobSidecarKeyLength = 48 // slot_to_rotating_buffer(blob.slot) ++ blob.slot ++ blob.block_root

// SaveBlobSidecar saves the blobs for a given epoch in the sidecar bucket. When we receive a blob:
//
//  1. Convert slot using a modulo operator to [0, maxSlots] where maxSlots = MAX_BLOB_EPOCHS*SLOTS_PER_EPOCH
//
//  2. Compute key for blob as bytes(slot_to_rotating_buffer(blob.slot)) ++ bytes(blob.slot) ++ blob.block_root
//
//  3. Begin the save algorithm:  If the incoming blob has a slot bigger than the saved slot at the spot
//     in the rotating keys buffer, we overwrite all elements for that slot.
func (s *Store) SaveBlobSidecar(ctx context.Context, blobSidecars *ethpb.BlobSidecars) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlobSidecar")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		encodedBlobSidecar, err := encode(ctx, blobSidecars)
		if err != nil {
			return err
		}
		bkt := tx.Bucket(blobsBucket)
		c := bkt.Cursor()
		newKey := blobSidecarKey(blobSidecars.Sidecars[0])
		rotatingBufferPrefix := newKey[0:8]
		var replacingKey []byte
		for k, _ := c.Seek(rotatingBufferPrefix); bytes.HasPrefix(k, rotatingBufferPrefix); k, _ = c.Next() {
			if len(k) != 0 {
				replacingKey = k
				oldSlotBytes := replacingKey[8:16]
				oldSlot := bytesutil.BytesToSlotBigEndian(oldSlotBytes)
				if oldSlot >= blobSidecars.Sidecars[0].Slot {
					return fmt.Errorf("attempted to save blob with slot %d but already have older blob with slot %d", blobSidecars.Sidecars[0].Slot, oldSlot)
				}
				break
			}
		}
		// If there is no element stored at blob.slot % MAX_SLOTS_TO_PERSIST_BLOBS, then we simply
		// store the blob by key and exit early.
		if len(replacingKey) == 0 {
			return bkt.Put(newKey, encodedBlobSidecar)
		}

		if err := bkt.Delete(replacingKey); err != nil {
			log.WithError(err).Warnf("Could not delete blob with key %#x", replacingKey)
		}
		return bkt.Put(newKey, encodedBlobSidecar)
	})
}

// BlobSidecarsByRoot retrieves the blobs given a beacon block root.
func (s *Store) BlobSidecarsByRoot(ctx context.Context, beaconBlockRoot [32]byte) (*ethpb.BlobSidecars, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlobSidecarsByRoot")
	defer span.End()

	var enc []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(blobsBucket).Cursor()
		// Bucket size is bounded and bolt cursors are fast. Moreover, a thin caching layer can be added.
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if len(k) != blobSidecarKeyLength {
				continue
			}
			if bytes.HasSuffix(k, beaconBlockRoot[:]) {
				enc = v
				break
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sidecars := &ethpb.BlobSidecars{}
	if err := decode(ctx, enc, sidecars); err != nil {
		return nil, err
	}
	return sidecars, nil
}

// BlobSidecarsBySlot retrieves sidecars from a slot.
func (s *Store) BlobSidecarsBySlot(ctx context.Context, slot types.Slot) (*ethpb.BlobSidecars, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlobSidecarsBySlot")
	defer span.End()

	var enc []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(blobsBucket).Cursor()
		// Bucket size is bounded and bolt cursors are fast. Moreover, a thin caching layer can be added.
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if len(k) != blobSidecarKeyLength {
				continue
			}
			slotInKey := bytesutil.BytesToSlotBigEndian(k[8:16])
			if slotInKey == slot {
				enc = v
				break
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sidecars := &ethpb.BlobSidecars{}
	if err := decode(ctx, enc, sidecars); err != nil {
		return nil, err
	}
	return sidecars, nil
}

// DeleteBlobSidecar returns true if the blobs are in the db.
func (s *Store) DeleteBlobSidecar(ctx context.Context, beaconBlockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteBlobSidecar")
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

// We define a blob sidecar key as: bytes(slot_to_rotating_buffer(blob.slot)) ++ bytes(blob.slot) ++ blob.block_root
// where slot_to_rotating_buffer(slot) = slot % MAX_SLOTS_TO_PERSIST_BLOBS.
func blobSidecarKey(blob *ethpb.BlobSidecar) []byte {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	maxEpochsToPersistBlobs := params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest
	maxSlotsToPersistBlobs := types.Slot(maxEpochsToPersistBlobs.Mul(uint64(slotsPerEpoch)))
	slotInRotatingBuffer := blob.Slot.ModSlot(maxSlotsToPersistBlobs)
	key := bytesutil.SlotToBytesBigEndian(slotInRotatingBuffer)
	key = append(key, bytesutil.SlotToBytesBigEndian(blob.Slot)...)
	key = append(key, blob.BlockRoot...)
	return key
}
