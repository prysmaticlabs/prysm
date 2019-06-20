package helpers

import (
	"fmt"
	"math/big"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// VoteHierarchyMap struct that holds all the relevant data in order to count and
// choose the best vote.
type VoteHierarchyMap struct {
	BestVote       *pb.Eth1Data
	bestVoteHeight *big.Int
	blockHeight    *big.Int
	mostVotes      uint64
	VoteCountMap   map[string]VoteHierarchy
}

// VoteHierarchy is a structure we use in order to count deposit votes and
// break ties between similarly voted deposits
type VoteHierarchy struct {
	votes    uint64
	height   *big.Int
	eth1Data *pb.Eth1Data
}

// EmptyVoteHierarchyMap creates and returns an empty VoteHierarchyMap.
func EmptyVoteHierarchyMap() *VoteHierarchyMap {
	vm := &VoteHierarchyMap{}
	vm.VoteCountMap = make(map[string]VoteHierarchy)
	return vm
}

// CountVote takes a votecount map and adds the given vote to it in the relevant
// position while updating the best vote, most votes and best vote hash.
func CountVote(voteMap *VoteHierarchyMap, vote *pb.Eth1Data, blockHeight *big.Int) (*VoteHierarchyMap, error) {
	he, err := ssz.HashedEncoding(vote)
	if err != nil {
		return &VoteHierarchyMap{}, fmt.Errorf("could not get encoded hash of eth1data object: %v", err)
	}
	v, ok := voteMap.VoteCountMap[string(he[:])]

	if !ok {
		v = VoteHierarchy{votes: 1, height: blockHeight, eth1Data: vote}
		voteMap.VoteCountMap[string(he[:])] = v
	} else {
		v.votes = v.votes + 1
		voteMap.VoteCountMap[string(he[:])] = v
	}
	if v.votes > voteMap.mostVotes {
		voteMap.mostVotes = v.votes
		voteMap.BestVote = vote
		voteMap.bestVoteHeight = blockHeight
	} else if v.votes == voteMap.mostVotes &&
		//breaking ties by favoring votes with higher block height.
		v.height.Cmp(voteMap.bestVoteHeight) == 1 {
		voteMap.BestVote = vote
		voteMap.bestVoteHeight = v.height
	}
	return voteMap, nil
}
