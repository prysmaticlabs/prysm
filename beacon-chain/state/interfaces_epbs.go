package state

import (
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
)

type ReadOnlyEpbsFields interface {
	IsParentBlockFull() (bool, error)
	ExecutionPayloadHeader() (*enginev1.ExecutionPayloadHeaderEPBS, error)
	LatestBlockHash() ([]byte, error)
	LatestFullSlot() (primitives.Slot, error)
	LastWithdrawalsRoot() ([]byte, error)
}

type WriteOnlyEpbsFields interface {
	SetExecutionPayloadHeader(val *enginev1.ExecutionPayloadHeaderEPBS) error
	SetLatestBlockHash(val []byte) error
	SetLatestFullSlot(val primitives.Slot) error
	SetLastWithdrawalsRoot(val []byte) error
}
