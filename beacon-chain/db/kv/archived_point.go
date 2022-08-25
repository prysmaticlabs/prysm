package kv

import (
	"context"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// LastArchivedSlot from the db.
func (s *Store) LastArchivedSlot(ctx context.Context) (types.Slot, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastArchivedSlot")
	defer span.End()
	var index types.Slot
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(stateSlotIndicesBucket)
		b, _ := bkt.Cursor().Last()
		index = bytesutil.BytesToSlotBigEndian(b)
		return nil
	})

	return index, err
}
