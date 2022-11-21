package light_client

import (
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
)

// TODO: maybe turn this into an interface so we don't have to copy from Finality/OptimisticUpdate
type Update struct {
	AttestedHeader          *ethpbv1.BeaconBlockHeader
	NextSyncCommittee       *ethpbv2.SyncCommittee
	NextSyncCommitteeBranch [][]byte
	FinalizedHeader         *ethpbv1.BeaconBlockHeader
	FinalityBranch          [][]byte
	SyncAggregate           *ethpbv1.SyncAggregate
	SignatureSlot           uint64
}

func (u *Update) IsSyncCommiteeUpdate() bool {
	// TODO: implement
	panic("not implemented")
}

func (u *Update) IsFinalityUpdate() bool {
	// TODO: implement
	panic("not implemented")
}

func (u *Update) IsBetterUpdate(newUpdate *Update) bool {
	// TODO: implement
	panic("not implemented")
}
