package kv

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func (k *Store) ArchivedActiveValidatorChanges(ctx context.Context, epoch uint64) (*ethpb.ArchivedActiveSetChanges, error) {
	return nil, errors.New("unimplemented")
}

func (k *Store) SaveArchivedActiveValidatorChanges(ctx context.Context, changes *ethpb.ArchivedActiveSetChanges) error {
	return errors.New("unimplemented")
}

func (k *Store) ArchivedCommitteeInfo(ctx context.Context, epoch uint64) (*ethpb.ArchivedCommitteeInfo, error) {
	return nil, errors.New("unimplemented")
}

func (k *Store) SaveArchivedCommitteeInfo(ctx context.Context, info *ethpb.ArchivedCommitteeInfo) error {
	return errors.New("unimplemented")
}

func (k *Store) ArchivedBalances(ctx context.Context, epoch uint64) ([]uint64, error) {
	return nil, errors.New("unimplemented")
}

func (k *Store) SaveArchivedBalances(ctx context.Context, balances []uint64) error {
	return errors.New("unimplemented")
}

func (k *Store) ArchivedActiveIndices(ctx context.Context, epoch uint64) ([]uint64, error) {
	return nil, errors.New("unimplemented")
}

func (k *Store) SaveArchivedActiveIndices(ctx context.Context, indices []uint64) error {
	return errors.New("unimplemented")
}

func (k *Store) ArchivedValidatorParticipation(ctx context.Context, epoch uint64) (*ethpb.ValidatorParticipation, error) {
	return nil, errors.New("unimplemented")
}

func (k *Store) SaveArchivedValidatorParticipation(ctx context.Context, part *ethpb.ValidatorParticipation) error {
	return errors.New("unimplemented")
}
