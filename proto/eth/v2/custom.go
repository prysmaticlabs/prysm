package eth

import (
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"math/bits"
)

const (
	NextSyncCommitteeIndex = uint64(55)
	FinalizedRootIndex     = uint64(105)
)

type Update interface {
	GetAttestedHeader() *ethpbv1.BeaconBlockHeader
	GetNextSyncCommittee() *SyncCommittee
	GetNextSyncCommitteeBranch() [][]byte
	GetFinalizedHeader() *ethpbv1.BeaconBlockHeader
	GetFinalityBranch() [][]byte
	GetSyncAggregate() *ethpbv1.SyncAggregate
	GetSignatureSlot() types.Slot
}

var _ Update = (*FinalityUpdate)(nil)

func (x *FinalityUpdate) GetNextSyncCommittee() *SyncCommittee {
	return &SyncCommittee{}
}

// TODO: move this somewhere common
func FloorLog2(x uint64) int {
	return bits.Len64(uint64(x - 1))
}

func (x *FinalityUpdate) GetNextSyncCommitteeBranch() [][]byte {
	return make([][]byte, FloorLog2(NextSyncCommitteeIndex))
}

var _ Update = (*OptimisticUpdate)(nil)

func (x *OptimisticUpdate) GetNextSyncCommittee() *SyncCommittee {
	return &SyncCommittee{}
}

func (x *OptimisticUpdate) GetNextSyncCommitteeBranch() [][]byte {
	return make([][]byte, FloorLog2(NextSyncCommitteeIndex))
}

func (x *OptimisticUpdate) GetFinalizedHeader() *ethpbv1.BeaconBlockHeader {
	return &ethpbv1.BeaconBlockHeader{}
}

func (x *OptimisticUpdate) GetFinalityBranch() [][]byte {
	return make([][]byte, FloorLog2(FinalizedRootIndex))
}
