package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// ArchivedActiveValidatorChanges retrieval by epoch.
func (k *Store) ArchivedActiveValidatorChanges(ctx context.Context, epoch uint64) (*ethpb.ArchivedActiveSetChanges, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.ArchivedActiveValidatorChanges")
	defer span.End()

	buf := uint64ToBytes(epoch)
	var changes *ethpb.ArchivedActiveSetChanges
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(archivedValidatorSetChangesBucket)
		enc := bkt.Get(buf)
		if enc == nil {
			return nil
		}
		changes = &ethpb.ArchivedActiveSetChanges{}
		return proto.Unmarshal(enc, changes)
	})
	return changes, err
}

// SaveArchivedActiveValidatorChanges by epoch.
func (k *Store) SaveArchivedActiveValidatorChanges(ctx context.Context, epoch uint64, changes *ethpb.ArchivedActiveSetChanges) error {
	return errors.New("unimplemented")
}

// ArchivedCommitteeInfo retrieval by epoch.
func (k *Store) ArchivedCommitteeInfo(ctx context.Context, epoch uint64) (*ethpb.ArchivedCommitteeInfo, error) {
	return nil, errors.New("unimplemented")
}

// SaveArchivedCommitteeInfo by epoch.
func (k *Store) SaveArchivedCommitteeInfo(ctx context.Context, epoch uint64, info *ethpb.ArchivedCommitteeInfo) error {
	return errors.New("unimplemented")
}

// ArchivedBalances retrieval by epoch.
func (k *Store) ArchivedBalances(ctx context.Context, epoch uint64) ([]uint64, error) {
	return nil, errors.New("unimplemented")
}

// SaveArchivedBalances by epoch.
func (k *Store) SaveArchivedBalances(ctx context.Context, epoch uint64, balances []uint64) error {
	return errors.New("unimplemented")
}

// ArchivedActiveIndices retrieval by epoch.
func (k *Store) ArchivedActiveIndices(ctx context.Context, epoch uint64) ([]uint64, error) {
	return nil, errors.New("unimplemented")
}

// SaveArchivedActiveIndices by epoch.
func (k *Store) SaveArchivedActiveIndices(ctx context.Context, epoch uint64, indices []uint64) error {
	return errors.New("unimplemented")
}

// ArchivedValidatorParticipation retrieval by epoch.
func (k *Store) ArchivedValidatorParticipation(ctx context.Context, epoch uint64) (*ethpb.ValidatorParticipation, error) {
	return nil, errors.New("unimplemented")
}

// SaveArchivedValidatorParticipation by epoch.
func (k *Store) SaveArchivedValidatorParticipation(ctx context.Context, epoch uint64, part *ethpb.ValidatorParticipation) error {
	return errors.New("unimplemented")
}
