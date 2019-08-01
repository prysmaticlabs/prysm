package blockchain

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var (
	reorgCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reorg_counter",
		Help: "The number of chain reorganization events that have happened in the fork choice rule",
	})
)
var blkAncestorCache = cache.NewBlockAncestorCache()

// TargetsFetcher defines a struct which can retrieve latest attestation targets
// from a given justified state.
type TargetsFetcher interface {
	AttestationTargets(justifiedState *pb.BeaconState) (map[uint64]*pb.AttestationTarget, error)
}

// AttestationTargets retrieves the list of attestation targets since last finalized epoch,
// each attestation target consists of validator index and its attestation target (i.e. the block
// which the validator attested to)
func (c *ChainService) AttestationTargets(state *pb.BeaconState) (map[uint64]*pb.AttestationTarget, error) {
	indices, err := helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state))
	if err != nil {
		return nil, err
	}

	attestationTargets := make(map[uint64]*pb.AttestationTarget)
	for i, index := range indices {
		target, err := c.attsService.LatestAttestationTarget(state, index)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve attestation target: %v", err)
		}
		if target == nil {
			continue
		}
		attestationTargets[uint64(i)] = target
	}
	return attestationTargets, nil
}
