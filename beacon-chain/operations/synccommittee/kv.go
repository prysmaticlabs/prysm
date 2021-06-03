package synccommittee

import (
	"sync"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var hashFn = hashutil.HashProto

// Store defines the caches for various sync committee objects
// such as signature(un-aggregated) and contribution(aggregated).
type Store struct {
	signatureLock     sync.RWMutex
	signatureCache    map[types.Slot][]*ethpb.SyncCommitteeSignature
	contributionLock  sync.RWMutex
	contributionCache map[types.Slot][]*ethpb.SyncCommitteeContribution
}

// NewStore initializes a new sync committee store.
func NewStore() *Store {
	return &Store{
		signatureCache:    make(map[types.Slot][]*ethpb.SyncCommitteeSignature),
		contributionCache: make(map[types.Slot][]*ethpb.SyncCommitteeContribution),
	}
}
