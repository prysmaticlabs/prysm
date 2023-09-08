package kv

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
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

// SaveBlobSidecar saves the blobs for a given epoch in the sidecar bucket. When we receive a blob:
//
//  1. Convert slot using a modulo operator to [0, maxSlots] where maxSlots = MAX_BLOB_EPOCHS*SLOTS_PER_EPOCH
//
//  2. Compute key for blob as bytes(slot_to_rotating_buffer(blob.slot)) ++ bytes(blob.slot) ++ blob.block_root
//
//  3. Begin the save algorithm:  If the incoming blob has a slot bigger than the saved slot at the spot
//     in the rotating keys buffer, we overwrite all elements for that slot. Otherwise, we merge the blob with an existing one.
func (s *Store) SaveBlobSidecar(ctx context.Context, scs []*ethpb.BlobSidecar) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlobSidecar")
	defer span.End()

	if len(scs) == 0 {
		return errEmptySidecar
	}
	newKey := blobSidecarKey(scs[0])
	rotatingBufferPrefix := newKey.BufferPrefix()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blobsBucket)
		c := bkt.Cursor()
		var replacingKey blobRotatingKey
		for k, _ := c.Seek(rotatingBufferPrefix); bytes.HasPrefix(k, rotatingBufferPrefix); k, _ = c.Next() {
			if len(k) != 0 {
				replacingKey = k
				break
			}
		}
		var existing []byte
		sc := &ethpb.BlobSidecars{}
		// If there is no element stored at blob.slot % MAX_SLOTS_TO_PERSIST_BLOBS, then we simply
		// store the blob by key and exit early.
		if len(replacingKey) != 0 {
			oldSlot := replacingKey.Slot()
			oldEpoch := slots.ToEpoch(oldSlot)
			// The blob we are replacing is too old, so we delete it.
			if slots.ToEpoch(scs[0].Slot) >= oldEpoch.Add(uint64(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest)) {
				if err := bkt.Delete(replacingKey); err != nil {
					log.WithError(err).Warnf("Could not delete blob with key %#x", replacingKey)
					return err
				}
			} else {
				// Otherwise, we need to merge the new blob with the old blob.
				existing = bkt.Get(replacingKey)
				if err := decode(ctx, existing, sc); err != nil {
					return err
				}
			}
		}

		sc.Sidecars = append(sc.Sidecars, scs...)
		sortSidecars(sc.Sidecars)
		var err error
		sc.Sidecars, err = validUniqueSidecars(sc.Sidecars)
		if err != nil {
			return err
		}
		updated, err := encode(ctx, sc)
		if err != nil {
			return err
		}

		// don't write if the merged result is the same as before
		if len(existing) == len(updated) && bytes.Equal(existing, updated) {
			return nil
		}

		return bkt.Put(newKey, updated)
	})
}

var (
	errBlobSlotMismatch     = errors.New("sidecar slot mismatch")
	errBlobParentMismatch   = errors.New("sidecar parent root mismatch")
	errBlobRootMismatch     = errors.New("sidecar root mismatch")
	errBlobProposerMismatch = errors.New("sidecar proposer index mismatch")
	errBlobSidecarLimit     = errors.New("sidecar exceeds maximum number of blobs")
	errEmptySidecar         = errors.New("nil or empty blob sidecars")
)

// validUniqueSidecars ensures that all sidecars have the same slot, parent root, block root, and proposer index, and no more than MAX_BLOB_EPOCHS.
func validUniqueSidecars(scs []*ethpb.BlobSidecar) ([]*ethpb.BlobSidecar, error) {
	if len(scs) == 0 {
		return nil, errEmptySidecar
	}

	// If there's only 1 sidecar, we've got nothing to compare.
	if len(scs) == 1 {
		return scs, nil
	}

	prev := scs[0]
	didx := 1
	for i := 1; i < len(scs); i++ {
		sc := scs[i]
		if sc.Slot != prev.Slot {
			return nil, errors.Wrapf(errBlobSlotMismatch, "%d != %d", sc.Slot, prev.Slot)
		}
		if !bytes.Equal(sc.BlockParentRoot, prev.BlockParentRoot) {
			return nil, errors.Wrapf(errBlobParentMismatch, "%x != %x", sc.BlockParentRoot, prev.BlockParentRoot)
		}
		if !bytes.Equal(sc.BlockRoot, prev.BlockRoot) {
			return nil, errors.Wrapf(errBlobRootMismatch, "%x != %x", sc.BlockRoot, prev.BlockRoot)
		}
		if sc.ProposerIndex != prev.ProposerIndex {
			return nil, errors.Wrapf(errBlobProposerMismatch, "%d != %d", sc.ProposerIndex, prev.ProposerIndex)
		}
		// skip duplicate
		if sc.Index == prev.Index {
			continue
		}
		if didx != i {
			scs[didx] = scs[i]
		}
		prev = scs[i]
		didx += 1
	}

	if didx > fieldparams.MaxBlobsPerBlock {
		return nil, errors.Wrapf(errBlobSidecarLimit, "%d > %d", didx, fieldparams.MaxBlobsPerBlock)
	}
	return scs[0:didx], nil
}

// sortSidecars sorts the sidecars by their index.
func sortSidecars(scs []*ethpb.BlobSidecar) {
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
func blobSidecarKey(blob *ethpb.BlobSidecar) blobRotatingKey {
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
		b := tx.Bucket(chainMetadataBucket)
		v := b.Get(blobRetentionEpochsKey)
		if v == nil {
			if err := b.Put(blobRetentionEpochsKey, bytesutil.Uint64ToBytesBigEndian(uint64(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest))); err != nil {
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
