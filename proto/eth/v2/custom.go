package eth

import (
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
)

const (
	NextSyncCommitteeIndex = uint64(55)
)

type Update interface {
	GetAttestedHeader() *ethpbv1.BeaconBlockHeader
	GetNextSyncCommittee() *SyncCommittee
	GetNextSyncCommitteeBranch() [][]byte
	GetFinalizedHeader() *ethpbv1.BeaconBlockHeader
	GetFinalityBranch() [][]byte
	GetSyncAggregate() *ethpbv1.SyncAggregate
	GetSignatureSlot() uint64
}

var _ Update = (*FinalityUpdate)(nil)

func (x *FinalityUpdate) GetNextSyncCommittee() *SyncCommittee {
	return &SyncCommittee{}
}

func (x *FinalityUpdate) GetNextSyncCommitteeBranch() [][]byte {
	return make([][]byte, NextSyncCommitteeIndex)
}
