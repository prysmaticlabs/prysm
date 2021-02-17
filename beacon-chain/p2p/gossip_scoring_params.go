package p2p

import (
	"math"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const (
	// beaconBlockWeight specifies the scoring weight that we apply to
	// our beacon block topic.
	beaconBlockWeight = 0.8
	// aggregateWeight specifies the scoring weight that we apply to
	// our aggregate topic.
	aggregateWeight = 0.5
	// attestationTotalWeight specifies the scoring weight that we apply to
	// our attestation subnet topic.
	attestationTotalWeight = 1
	// decayToZero specifies the terminal value that we will use when decaying
	// a value.
	decayToZero = 0.01
)

func peerScoringParams() (*pubsub.PeerScoreParams, *pubsub.PeerScoreThresholds) {
	thresholds := &pubsub.PeerScoreThresholds{
		GossipThreshold:             -4000,
		PublishThreshold:            -8000,
		GraylistThreshold:           -16000,
		AcceptPXThreshold:           100,
		OpportunisticGraftThreshold: 5,
	}
	scoreParams := &pubsub.PeerScoreParams{
		Topics:        make(map[string]*pubsub.TopicScoreParams),
		TopicScoreCap: 32.72,
		AppSpecificScore: func(p peer.ID) float64 {
			return 0
		},
		AppSpecificWeight:           1,
		IPColocationFactorWeight:    -35.11,
		IPColocationFactorThreshold: 10,
		IPColocationFactorWhitelist: nil,
		BehaviourPenaltyWeight:      -15.92,
		BehaviourPenaltyThreshold:   6,
		BehaviourPenaltyDecay:       scoreDecay(10 * oneEpochDuration()),
		DecayInterval:               1 * oneSlotDuration(),
		DecayToZero:                 decayToZero,
		RetainScore:                 100 * oneEpochDuration(),
	}
	return scoreParams, thresholds
}

func topicScoreParams(topic string) *pubsub.TopicScoreParams {
	switch {
	case strings.Contains(topic, "beacon_block"):
		return defaultBlockTopicParams()
	case strings.Contains(topic, "beacon_aggregate_and_proof"):
		return defaultAggregateTopicParams()
	case strings.Contains(topic, "beacon_attestation"):
		return defaultAggregateSubnetTopicParams()
	default:
		return nil
	}
}

// Based on Ben's tested parameters for lighthouse.
// https://gist.github.com/blacktemplar/5c1862cb3f0e32a1a7fb0b25e79e6e2c

func defaultBlockTopicParams() *pubsub.TopicScoreParams {
	decayEpoch := time.Duration(5)
	blocksPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	return &pubsub.TopicScoreParams{
		TopicWeight:                     beaconBlockWeight,
		TimeInMeshWeight:                0.0324,
		TimeInMeshQuantum:               1 * oneSlotDuration(),
		TimeInMeshCap:                   300,
		FirstMessageDeliveriesWeight:    1,
		FirstMessageDeliveriesDecay:     scoreDecay(20 * oneEpochDuration()),
		FirstMessageDeliveriesCap:       23,
		MeshMessageDeliveriesWeight:     -0.717,
		MeshMessageDeliveriesDecay:      scoreDecay(decayEpoch * oneEpochDuration()),
		MeshMessageDeliveriesCap:        float64(blocksPerEpoch * uint64(decayEpoch)),
		MeshMessageDeliveriesThreshold:  float64(blocksPerEpoch*uint64(decayEpoch)) / 10,
		MeshMessageDeliveriesWindow:     2 * time.Second,
		MeshMessageDeliveriesActivation: 4 * oneEpochDuration(),
		MeshFailurePenaltyWeight:        -0.717,
		MeshFailurePenaltyDecay:         scoreDecay(decayEpoch * oneEpochDuration()),
		InvalidMessageDeliveriesWeight:  -140.4475,
		InvalidMessageDeliveriesDecay:   scoreDecay(50 * oneEpochDuration()),
	}
}

func defaultAggregateTopicParams() *pubsub.TopicScoreParams {
	aggPerEpoch := aggregatorsPerSlot() * uint64(params.BeaconConfig().SlotsPerEpoch)
	return &pubsub.TopicScoreParams{
		TopicWeight:                     aggregateWeight,
		TimeInMeshWeight:                0.0324,
		TimeInMeshQuantum:               1 * oneSlotDuration(),
		TimeInMeshCap:                   300,
		FirstMessageDeliveriesWeight:    0.128,
		FirstMessageDeliveriesDecay:     scoreDecay(1 * oneEpochDuration()),
		FirstMessageDeliveriesCap:       179,
		MeshMessageDeliveriesWeight:     -0.064,
		MeshMessageDeliveriesDecay:      scoreDecay(1 * oneEpochDuration()),
		MeshMessageDeliveriesCap:        float64(aggPerEpoch),
		MeshMessageDeliveriesThreshold:  float64(aggPerEpoch / 50),
		MeshMessageDeliveriesWindow:     2 * time.Second,
		MeshMessageDeliveriesActivation: 32 * oneSlotDuration(),
		MeshFailurePenaltyWeight:        -0.064,
		MeshFailurePenaltyDecay:         scoreDecay(1 * oneEpochDuration()),
		InvalidMessageDeliveriesWeight:  -140.4475,
		InvalidMessageDeliveriesDecay:   scoreDecay(50 * oneEpochDuration()),
	}
}

func defaultAggregateSubnetTopicParams() *pubsub.TopicScoreParams {
	topicWeight := attestationTotalWeight / float64(params.BeaconNetworkConfig().AttestationSubnetCount)
	subnetWeight := activeValidators() / params.BeaconNetworkConfig().AttestationSubnetCount
	minimumWeight := subnetWeight / 50
	numPerSlot := time.Duration(subnetWeight / uint64(params.BeaconConfig().SlotsPerEpoch))
	comsPerSlot := committeeCountPerSlot()
	exceedsThreshold := comsPerSlot >= 2*params.BeaconNetworkConfig().AttestationSubnetCount/uint64(params.BeaconConfig().SlotsPerEpoch)
	firstDecay := time.Duration(1)
	meshDecay := time.Duration(4)
	if exceedsThreshold {
		firstDecay = 4
		meshDecay = 16
	}
	return &pubsub.TopicScoreParams{
		TopicWeight:                     topicWeight,
		TimeInMeshWeight:                0.0324,
		TimeInMeshQuantum:               numPerSlot,
		TimeInMeshCap:                   300,
		FirstMessageDeliveriesWeight:    0.955,
		FirstMessageDeliveriesDecay:     scoreDecay(firstDecay * oneEpochDuration()),
		FirstMessageDeliveriesCap:       24,
		MeshMessageDeliveriesWeight:     -37.55,
		MeshMessageDeliveriesDecay:      scoreDecay(meshDecay * oneEpochDuration()),
		MeshMessageDeliveriesCap:        float64(subnetWeight),
		MeshMessageDeliveriesThreshold:  float64(minimumWeight),
		MeshMessageDeliveriesWindow:     2 * time.Second,
		MeshMessageDeliveriesActivation: 17 * oneSlotDuration(),
		MeshFailurePenaltyWeight:        -37.55,
		MeshFailurePenaltyDecay:         scoreDecay(meshDecay * oneEpochDuration()),
		InvalidMessageDeliveriesWeight:  -4544,
		InvalidMessageDeliveriesDecay:   scoreDecay(50 * oneEpochDuration()),
	}
}

func oneSlotDuration() time.Duration {
	return time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second
}

func oneEpochDuration() time.Duration {
	return time.Duration(params.BeaconConfig().SlotsPerEpoch) * oneSlotDuration()
}

func scoreDecay(totalDurationDecay time.Duration) float64 {
	numOfTimes := totalDurationDecay / oneSlotDuration()
	return math.Pow(decayToZero, 1/float64(numOfTimes))
}

// Default to the min-genesis for the current moment, as p2p service
// has no access to the chain service.
func activeValidators() uint64 {
	return params.BeaconConfig().MinGenesisActiveValidatorCount
}

func committeeCountPerSlot() uint64 {
	// Use a static parameter for now rather than a dynamic one, we can use
	// the actual parameter later when we have figured out how to fix a circular
	// dependency in service startup order.
	return helpers.SlotCommitteeCount(activeValidators())
}

// Uses a very rough gauge for total aggregator size per slot.
func aggregatorsPerSlot() uint64 {
	comms := committeeCountPerSlot()
	totalAggs := comms * params.BeaconConfig().TargetAggregatorsPerCommittee
	return totalAggs
}
