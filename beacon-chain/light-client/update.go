package light_client

import (
	"bytes"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"math/bits"
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

func floorLog2(x uint64) int {
	return bits.Len64(uint64(x - 1))
}

func isEmptyWithLength(bb [][]byte, length uint64) bool {
	l := floorLog2(length)
	if len(bb) != l {
		return false
	}
	for _, b := range bb {
		if !bytes.Equal(b, []byte{}) {
			return false
		}
	}
	return true
}

func (u *Update) IsSyncCommiteeUpdate() bool {
	return !isEmptyWithLength(u.NextSyncCommitteeBranch, nextSyncCommitteeIndex)
}

func (u *Update) IsFinalityUpdate() bool {
	return !isEmptyWithLength(u.NextSyncCommitteeBranch, finalizedRootIndex)
}

func (u *Update) IsBetterUpdate(newUpdate *Update) bool {
	// TODO: implement
	panic("not implemented")
}
