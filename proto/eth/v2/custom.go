package eth

import (
	"bytes"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"math/bits"
)

const (
	CurrentSyncCommitteeIndex = uint64(54)
	NextSyncCommitteeIndex    = uint64(55)
	FinalizedRootIndex        = uint64(105)
)

type Update interface {
	GetAttestedHeader() *ethpbv1.BeaconBlockHeader
	GetNextSyncCommittee() *SyncCommittee
	GetNextSyncCommitteeBranch() [][]byte
	GetFinalizedHeader() *ethpbv1.BeaconBlockHeader
	SetFinalizedHeader(*ethpbv1.BeaconBlockHeader)
	GetFinalityBranch() [][]byte
	GetSyncAggregate() *ethpbv1.SyncAggregate
	GetSignatureSlot() types.Slot
}

// TODO: move this somewhere common
func FloorLog2(x uint64) int {
	return bits.Len64(uint64(x - 1))
}

var _ Update = (*FinalityUpdate)(nil)

func (x *FinalityUpdate) GetNextSyncCommittee() *SyncCommittee {
	return &SyncCommittee{}
}

func (x *FinalityUpdate) GetNextSyncCommitteeBranch() [][]byte {
	return make([][]byte, FloorLog2(NextSyncCommitteeIndex))
}

func (x *FinalityUpdate) SetFinalizedHeader(header *ethpbv1.BeaconBlockHeader) {
	x.FinalizedHeader = header
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

func (x *OptimisticUpdate) SetFinalizedHeader(header *ethpbv1.BeaconBlockHeader) {}

func (x *SyncCommittee) Equals(other *SyncCommittee) bool {
	if len(x.Pubkeys) != len(other.Pubkeys) {
		return false
	}
	for i := range x.Pubkeys {
		if !bytes.Equal(x.Pubkeys[i], other.Pubkeys[i]) {
			return false
		}
	}
	return bytes.Equal(x.AggregatePubkey, other.AggregatePubkey)
}
