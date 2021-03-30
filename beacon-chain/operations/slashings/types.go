package slashings

import (
	"context"
	"sync"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
)

// PoolManager maintains a pool of pending and recently included attester and proposer slashings.
// This pool is used by proposers to insert data into new blocks.
type PoolManager interface {
	PendingAttesterSlashings(ctx context.Context, state iface.ReadOnlyBeaconState, noLimit bool) []*ethpb.AttesterSlashing
	PendingProposerSlashings(ctx context.Context, state iface.ReadOnlyBeaconState, noLimit bool) []*ethpb.ProposerSlashing
	InsertAttesterSlashing(
		ctx context.Context,
		state iface.ReadOnlyBeaconState,
		slashing *ethpb.AttesterSlashing,
	) error
	InsertProposerSlashing(
		ctx context.Context,
		state iface.BeaconState,
		slashing *ethpb.ProposerSlashing,
	) error
	MarkIncludedAttesterSlashing(as *ethpb.AttesterSlashing)
	MarkIncludedProposerSlashing(ps *ethpb.ProposerSlashing)
}

// Pool is a concrete implementation of PoolManager.
type Pool struct {
	lock                    sync.RWMutex
	pendingProposerSlashing []*ethpb.ProposerSlashing
	pendingAttesterSlashing []*PendingAttesterSlashing
	included                map[types.ValidatorIndex]bool
}

// PendingAttesterSlashing represents an attester slashing in the operation pool.
// Allows for easy binary searching of included validator indexes.
type PendingAttesterSlashing struct {
	attesterSlashing *ethpb.AttesterSlashing
	validatorToSlash types.ValidatorIndex
}
