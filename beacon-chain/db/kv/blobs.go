package kv

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// DeleteBlobsSidecar removes the blobs from the db.
func (s *Store) DeleteBlobsSidecar(ctx context.Context, root [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteBlobsSidecar")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(blobsBucket).Delete(root[:])
	})
}

// SaveBlobsSidecar saves the blobs for a given epoch in the sidecar bucket.
func (s *Store) SaveBlobsSidecar(ctx context.Context, blob *ethpb.BlobsSidecar) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlobsSidecar")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blobsBucket)
		enc, err := encode(ctx, blob)
		if err != nil {
			return err
		}
		return bkt.Put(blob.BeaconBlockRoot, enc)
	})
}

// BlobsSidecar retrieves the blobs given a block root.
func (s *Store) BlobsSidecar(ctx context.Context, blockRoot [32]byte) (*ethpb.BlobsSidecar, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlobsSidecar")
	defer span.End()

	var enc []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		enc = tx.Bucket(blobsBucket).Get(blockRoot[:])
		return nil
	}); err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	blob := &ethpb.BlobsSidecar{}
	if err := decode(ctx, enc, blob); err != nil {
		return nil, err
	}
	return blob, nil
}

// BlobsSidecar retrieves sidecars from a slot.
func (s *Store) BlobsSidecarsBySlot(ctx context.Context, slot types.Slot) ([]*ethpb.BlobsSidecar, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.BlobsSidecarsBySlot")
	defer span.End()

	var blobsSidecars []*ethpb.BlobsSidecar
	err := s.db.View(func(tx *bolt.Tx) error {
		blockRoots, err := blockRootsBySlot(ctx, tx, slot)
		if err != nil {
			return err
		}

		for _, blockRoot := range blockRoots {
			enc := tx.Bucket(blobsBucket).Get(blockRoot[:])
			if len(enc) == 0 {
				return nil
			}
			blobs := &ethpb.BlobsSidecar{}
			if err := decode(ctx, enc, blobs); err != nil {
				return err
			}
			blobsSidecars = append(blobsSidecars, blobs)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve blobs")
	}
	return blobsSidecars, nil
}

func (s *Store) HasBlobsSidecar(ctx context.Context, root [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasBlobsSidecar")
	defer span.End()

	exists := false
	if err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blobsBucket)
		exists = bkt.Get(root[:]) != nil
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}
	return exists
}
