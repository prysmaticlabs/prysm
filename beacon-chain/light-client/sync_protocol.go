// Package light_client implements the light client for the Ethereum 2.0 Beacon Chain.
// It is based on the Altair light client spec at this revision:
// https://github.com/ethereum/consensus-specs/tree/208da34ac4e75337baf79adebf036ab595e39f15/specs/altair/light-client
package light_client

import (
	"bytes"
	"github.com/prysmaticlabs/prysm/v3/container/trie"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"math"
	"math/bits"
)

const (
	finalizedRootIndex        = uint64(105)
	currentSyncCommitteeIndex = uint64(54)
	nextSyncCommitteeIndex    = uint64(55)
)

// TODO: move this to prysm and implement
type LightClientUpdate struct{}

type lightClientStore struct {
	finalizedHeader               *ethpbv1.BeaconBlockHeader
	currentSyncCommittee          *ethpbv2.SyncCommittee
	nextSyncCommittee             *ethpbv2.SyncCommittee
	bestValidUpdate               *LightClientUpdate
	optimisticHeader              *ethpbv1.BeaconBlockHeader
	previousMaxActiveParticipants uint64
	currentMaxActiveParticipants  uint64
}

func floorLog2(x uint64) int {
	return bits.Len64(uint64(x - 1))
}

func getSubtreeIndex(index uint64) uint64 {
	return index % uint64(math.Pow(2, float64(floorLog2(index-1))))
}

func initializeLightClientStore(trustedBlockRoot [32]byte,
	bootstrap ethpbv2.Bootstrap) *lightClientStore {
	bootstrapRoot, err := bootstrap.Header.HashTreeRoot()
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(trustedBlockRoot[:], bootstrapRoot[:]) {
		panic("trusted block root does not match bootstrap header")
	}
	v1alpha1Committee := ethpb.SyncCommittee{
		Pubkeys:         bootstrap.CurrentSyncCommittee.GetPubkeys(),
		AggregatePubkey: bootstrap.CurrentSyncCommittee.GetAggregatePubkey(),
	}
	syncCommitteeRoot, err := v1alpha1Committee.HashTreeRoot()
	if !trie.VerifyMerkleProofWithDepth(
		bootstrap.Header.StateRoot,
		syncCommitteeRoot[:],
		getSubtreeIndex(currentSyncCommitteeIndex),
		bootstrap.CurrentSyncCommitteeBranch,
		uint64(floorLog2(currentSyncCommitteeIndex))) {
		panic("current sync committee merkle proof is invalid")
	}
	return &lightClientStore{
		finalizedHeader:      bootstrap.Header,
		currentSyncCommittee: bootstrap.CurrentSyncCommittee,
		nextSyncCommittee:    &ethpbv2.SyncCommittee{},
		optimisticHeader:     bootstrap.Header,
	}
}
