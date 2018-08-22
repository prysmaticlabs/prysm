package casper

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestGetIndicesForHeight(t *testing.T) {
	state := types.NewCrystallizedState(&pb.CrystallizedState{
		LastStateRecalc: 1,
		IndicesForHeights: []*pb.ShardAndCommitteeArray{
			{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{ShardId: 1, Committee: []uint32{0, 1, 2, 3, 4}},
				{ShardId: 2, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
			{ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{ShardId: 3, Committee: []uint32{0, 1, 2, 3, 4}},
				{ShardId: 4, Committee: []uint32{5, 6, 7, 8, 9}},
			}},
		}})
	if _, err := GetIndicesForHeight(state, 1000); err == nil {
		t.Error("getIndicesForHeight should have failed with invalid height")
	}
	committee, err := GetIndicesForHeight(state, 1)
	if err != nil {
		t.Errorf("getIndicesForHeight failed: %v", err)
	}
	if committee.ArrayShardAndCommittee[0].ShardId != 1 {
		t.Errorf("getIndicesForHeight returns shardID should be 1, got: %v", committee.ArrayShardAndCommittee[0].ShardId)
	}
	committee, _ = GetIndicesForHeight(state, 2)
	if committee.ArrayShardAndCommittee[0].ShardId != 3 {
		t.Errorf("getIndicesForHeight returns shardID should be 3, got: %v", committee.ArrayShardAndCommittee[0].ShardId)
	}
}
