package slashings

import (
	"sync"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// Pool implements a struct to maintain pending and recently included attester and
// proposer slashings. This pool is used by proposers to insert into new blocks.
type Pool struct {
	lock                    sync.RWMutex
	pendingProposerSlashing []*ethpb.ProposerSlashing
	pendingAttesterSlashing []*PendingAttesterSlashing
	included                map[uint64]bool
}

// PendingAttesterSlashing represents an attester slashing in the operation pool.
// Allows for easy binary searching of included validator indexes.
type PendingAttesterSlashing struct {
	attesterSlashing *ethpb.AttesterSlashing
	validatorToSlash uint64
}
