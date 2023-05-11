package kv

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SaveBlobSidecar saves the blobs for a given epoch in the sidecar bucket. When we receive a blob:
//
//  1. Convert slot using a modulo operator to [0, maxSlots] where maxSlots = MAX_BLOB_EPOCHS*SLOTS_PER_EPOCH
//
//  2. Compute key for blob as bytes(slot_to_rotating_buffer(blob.slot)) ++ bytes(blob.slot) ++ blob.block_root
//
//  3. Begin the save algorithm:  If the incoming blob has a slot bigger than the saved slot at the spot
//     in the rotating keys buffer, we overwrite all elements for that slot.
func (s *Store) SaveBlobSidecar(ctx context.Context, scs []*ethpb.BlobSidecar) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlobSidecar")
	defer span.End()

	sortSideCars(scs)
	if err := s.verifySideCars(scs); err != nil {
		return err
	}

	slot := scs[0].Slot
	return s.db.Update(func(tx *bolt.Tx) error {
		encodedBlobSidecar, err := encode(ctx, &ethpb.BlobSidecars{Sidecars: scs})
		if err != nil {
			return err
		}
		bkt := tx.Bucket(blobsBucket)
		c := bkt.Cursor()
		newKey := blobSidecarKey(scs[0])
		rotatingBufferPrefix := newKey[0:8]
		var replacingKey []byte
		for k, _ := c.Seek(rotatingBufferPrefix); bytes.HasPrefix(k, rotatingBufferPrefix); k, _ = c.Next() {
			if len(k) != 0 {
				replacingKey = k
				oldSlotBytes := replacingKey[8:16]
				oldSlot := bytesutil.BytesToSlotBigEndian(oldSlotBytes)
				if oldSlot >= slot {
					return fmt.Errorf("attempted to save blob with slot %d but already have older blob with slot %d", slot, oldSlot)
				}
				break
			}
		}
		// If there is no element stored at blob.slot % MAX_SLOTS_TO_PERSIST_BLOBS, then we simply
		// store the blob by key and exit early.
		if len(replacingKey) != 0 {
			if err := bkt.Delete(replacingKey); err != nil {
				log.WithError(err).Warnf("Could not delete blob with key %#x", replacingKey)
			}
		}
		return bkt.Put(newKey, encodedBlobSidecar)
	})
}

// verifySideCars ensures that all sidecars have the same slot, parent root, block root, and proposer index.
// It also ensures that indices are sequential and start at 0 and no more than MAX_BLOB_EPOCHS.
func (s *Store) verifySideCars(scs []*ethpb.BlobSidecar) error {
	if len(scs) == 0 {
		return errors.New("nil or empty blob sidecars")
	}
	if uint64(len(scs)) > params.BeaconConfig().MaxBlobsPerBlock {
		return fmt.Errorf("too many sidecars: %d > %d", len(scs), params.BeaconConfig().MaxBlobsPerBlock)
	}

	sl := scs[0].Slot
	pr := scs[0].BlockParentRoot
	r := scs[0].BlockRoot
	p := scs[0].ProposerIndex

	for i, sc := range scs {
		if sc.Slot != sl {
			return fmt.Errorf("sidecar slot mismatch: %d != %d", sc.Slot, sl)
		}
		if !bytes.Equal(sc.BlockParentRoot, pr) {
			return fmt.Errorf("sidecar parent root mismatch: %x != %x", sc.BlockParentRoot, pr)
		}
		if !bytes.Equal(sc.BlockRoot, r) {
			return fmt.Errorf("sidecar root mismatch: %x != %x", sc.BlockRoot, r)
		}
		if sc.ProposerIndex != p {
			return fmt.Errorf("sidecar proposer index mismatch: %d != %d", sc.ProposerIndex, p)
		}
		if sc.Index != uint64(i) {
			return fmt.Errorf("sidecar index mismatch: %d != %d", sc.Index, i)
		}
	}
	return nil
}

// sortSideCars sorts the sidecars by their index.
func sortSideCars(scs []*ethpb.BlobSidecar) {
	sort.Slice(scs, func(i, j int) bool {
		return scs[i].Index < scs[j].Index
	})
}

// BlobSidecarsByRoot retrieves the blobs for the given beacon block root.
// If the `indices` argument is omitted, all blobs for the root will be returned.
// Otherwise, the result will be filtered to only include the specified indices.
// An error will result if an invalid index is specified.
// The bucket size is bounded by 131072 entries. That's the most blobs a node will keep before rotating it out.
func (s *Store) BlobSidecarsByRoot(ctx context.Context, root [32]byte, indices ...uint64) ([]*ethpb.BlobSidecar, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlobSidecarsByRoot")
	defer span.End()

	var enc []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(blobsBucket).Cursor()
		// Bucket size is bounded and bolt cursors are fast. Moreover, a thin caching layer can be added.
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if bytes.HasSuffix(k, root[:]) {
				enc = v
				break
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if enc == nil {
		return nil, ErrNotFound
	}
	sc := &ethpb.BlobSidecars{}
	if err := decode(ctx, enc, sc); err != nil {
		return nil, err
	}

	return filterForIndices(sc, indices...)
}

func filterForIndices(sc *ethpb.BlobSidecars, indices ...uint64) ([]*ethpb.BlobSidecar, error) {
	if len(indices) == 0 {
		return sc.Sidecars, nil
	}
	// This loop assumes that the BlobSidecars value stores the complete set of blobs for a block
	// in ascending order from eg 0..3, without gaps. This allows us to assume the indices argument
	// maps 1:1 with indices in the BlobSidecars storage object.
	maxIdx := uint64(len(sc.Sidecars)) - 1
	sidecars := make([]*ethpb.BlobSidecar, len(indices))
	for i, idx := range indices {
		if idx > maxIdx {
			return nil, errors.Wrapf(ErrNotFound, "BlobSidecars missing index: index %d", idx)
		}
		sidecars[i] = sc.Sidecars[idx]
	}
	return sidecars, nil
}

// BlobSidecarsBySlot retrieves BlobSidecars for the given slot.
// If the `indices` argument is omitted, all blobs for the root will be returned.
// Otherwise, the result will be filtered to only include the specified indices.
// An error will result if an invalid index is specified.
// The bucket size is bounded by 131072 entries. That's the most blobs a node will keep before rotating it out.
func (s *Store) BlobSidecarsBySlot(ctx context.Context, slot types.Slot, indices ...uint64) ([]*ethpb.BlobSidecar, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlobSidecarsBySlot")
	defer span.End()

	var enc []byte
	sk := slotKey(slot)
	if err := s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(blobsBucket).Cursor()
		// Bucket size is bounded and bolt cursors are fast. Moreover, a thin caching layer can be added.
		for k, v := c.Seek(sk); bytes.HasPrefix(k, sk); k, _ = c.Next() {
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
	if enc == nil {
		return nil, ErrNotFound
	}
	sc := &ethpb.BlobSidecars{}
	if err := decode(ctx, enc, sc); err != nil {
		return nil, err
	}

	return filterForIndices(sc, indices...)
}

// DeleteBlobSidecar returns true if the blobs are in the db.
func (s *Store) DeleteBlobSidecar(ctx context.Context, beaconBlockRoot [32]byte) error {
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

// We define a blob sidecar key as: bytes(slot_to_rotating_buffer(blob.slot)) ++ bytes(blob.slot) ++ blob.block_root
// where slot_to_rotating_buffer(slot) = slot % MAX_SLOTS_TO_PERSIST_BLOBS.
func blobSidecarKey(blob *ethpb.BlobSidecar) []byte {
	key := slotKey(blob.Slot)
	key = append(key, bytesutil.SlotToBytesBigEndian(blob.Slot)...)
	key = append(key, blob.BlockRoot...)
	return key
}

func slotKey(slot types.Slot) []byte {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	maxEpochsToPersistBlobs := params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest
	maxSlotsToPersistBlobs := types.Slot(maxEpochsToPersistBlobs.Mul(uint64(slotsPerEpoch)))
	return bytesutil.SlotToBytesBigEndian(slot.ModSlot(maxSlotsToPersistBlobs))
}

func checkEpochsForBlobSidecarsRequestBucket(db *bolt.DB) error {
	if err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(epochsForBlobSidecarsRequestBucket)
		k := []byte("epoch-key")
		v := b.Get(k)
		if v == nil {
			if err := b.Put(k, bytesutil.Uint64ToBytesBigEndian(uint64(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest))); err != nil {
				return err
			}
			return nil
		}
		e := bytesutil.BytesToUint64BigEndian(v)
		if e != uint64(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest) {
			return fmt.Errorf("epochs for blobs request value in DB %d does not match config value %d", e, params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
