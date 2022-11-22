package light_client

import (
	"bytes"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"math/bits"
)

type Update struct {
	ethpbv2.Update
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
	return !isEmptyWithLength(u.GetNextSyncCommitteeBranch(), ethpbv2.NextSyncCommitteeIndex)
}

func (u *Update) IsFinalityUpdate() bool {
	return !isEmptyWithLength(u.GetNextSyncCommitteeBranch(), finalizedRootIndex)
}

func (u *Update) IsBetterUpdate(newUpdate *Update) bool {
	maxActiveParticipants := uint64(len(newUpdate.GetSyncAggregate().SyncCommitteeBits))
	newNumActiveParticipants := newUpdate.GetSyncAggregate().SyncCommitteeBits.Count()
	oldNumActiveParticipants := u.GetSyncAggregate().SyncCommitteeBits.Count()
	_ = newNumActiveParticipants*3 >= maxActiveParticipants*2
	_ = oldNumActiveParticipants*3 >= maxActiveParticipants*2
	// TODO: resume here
	return false
}
