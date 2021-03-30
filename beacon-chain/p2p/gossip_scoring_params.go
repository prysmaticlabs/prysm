package p2p

import (
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
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
	// attesterSlashingWeight specifies the scoring weight that we apply to
	// our attester slashing topic.
	attesterSlashingWeight = 0.05
	// proposerSlashingWeight specifies the scoring weight that we apply to
	// our proposer slashing topic.
	proposerSlashingWeight = 0.05
	// voluntaryExitWeight specifies the scoring weight that we apply to
	// our voluntary exit topic.
	voluntaryExitWeight = 0.05

	// maxInMeshScore describes the max score a peer can attain from being in the mesh.
	maxInMeshScore = 10
	// maxFirstDeliveryScore describes the max score a peer can obtain from first deliveries.
	maxFirstDeliveryScore = 40

	// decayToZero specifies the terminal value that we will use when decaying
	// a value.
	decayToZero = 0.01

	// dampeningFactor reduces the amount by which the various thresholds and caps are created.
	dampeningFactor = 90

	// gossipSubD is a hardcoded value representing the D gossip paramter, which signifies the
	// degree of the gossip mesh.
	gossipSubD = 8
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

func (s *Service) topicScoreParams(topic string) (*pubsub.TopicScoreParams, error) {
	activeValidators, err := s.retrieveActiveValidators()
	if err != nil {
		return nil, err
	}
	switch {
	case strings.Contains(topic, "beacon_block"):
		return defaultBlockTopicParams(), nil
	case strings.Contains(topic, "beacon_aggregate_and_proof"):
		return defaultAggregateTopicParams(activeValidators), nil
	case strings.Contains(topic, "beacon_attestation"):
		return defaultAggregateSubnetTopicParams(activeValidators), nil
	case strings.Contains(topic, "voluntary_exit"):
		return defaultVoluntaryExitTopicParams(), nil
	case strings.Contains(topic, "proposer_slashing"):
		return defaultProposerSlashingTopicParams(), nil
	case strings.Contains(topic, "attester_slashing"):
		return defaultAttesterSlashingTopicParams(), nil
	default:
		return nil, errors.Errorf("unrecognized topic provided for parameter registration: %s", topic)
	}
}

func (s *Service) retrieveActiveValidators() (uint64, error) {
	rt := s.cfg.DB.LastArchivedRoot(s.ctx)
	if rt == params.BeaconConfig().ZeroHash {
		genState, err := s.cfg.DB.GenesisState(s.ctx)
		if err != nil {
			return 0, err
		}
		if genState == nil {
			return 0, errors.New("no genesis state exists")
		}
		return helpers.ActiveValidatorCount(genState, helpers.CurrentEpoch(genState))
	}
	bState, err := s.cfg.DB.State(s.ctx, rt)
	if err != nil {
		return 0, err
	}
	if bState == nil {
		return 0, errors.Errorf("no state with root %#x exists", rt)
	}
	return helpers.ActiveValidatorCount(bState, helpers.CurrentEpoch(bState))
}

// Based on Ben's tested parameters for lighthouse.
// https://gist.github.com/blacktemplar/5c1862cb3f0e32a1a7fb0b25e79e6e2c

func defaultBlockTopicParams() *pubsub.TopicScoreParams {
	decayEpoch := time.Duration(5)
	blocksPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	return &pubsub.TopicScoreParams{
		TopicWeight:                     beaconBlockWeight,
		TimeInMeshWeight:                maxInMeshScore / inMeshCap(),
		TimeInMeshQuantum:               inMeshTime(),
		TimeInMeshCap:                   inMeshCap(),
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

func defaultAggregateTopicParams(activeValidators uint64) *pubsub.TopicScoreParams {
	aggPerSlot := aggregatorsPerSlot(activeValidators)
	firstMessageCap := decay(scoreDecay(1*oneEpochDuration()), float64(aggPerSlot*2/gossipSubD))
	firstMessageWeight := maxFirstDeliveryScore / firstMessageCap
	meshThreshold := decayThreshold(scoreDecay(1*oneEpochDuration()), float64(aggPerSlot)/dampeningFactor)
	meshWeight := -maxScore() / (aggregateWeight * meshThreshold * meshThreshold)
	meshCap := 4 * meshThreshold
	return &pubsub.TopicScoreParams{
		TopicWeight:                     aggregateWeight,
		TimeInMeshWeight:                maxInMeshScore / inMeshCap(),
		TimeInMeshQuantum:               inMeshTime(),
		TimeInMeshCap:                   inMeshCap(),
		FirstMessageDeliveriesWeight:    firstMessageWeight,
		FirstMessageDeliveriesDecay:     scoreDecay(1 * oneEpochDuration()),
		FirstMessageDeliveriesCap:       firstMessageCap,
		MeshMessageDeliveriesWeight:     meshWeight,
		MeshMessageDeliveriesDecay:      scoreDecay(1 * oneEpochDuration()),
		MeshMessageDeliveriesCap:        meshCap,
		MeshMessageDeliveriesThreshold:  meshThreshold,
		MeshMessageDeliveriesWindow:     2 * time.Second,
		MeshMessageDeliveriesActivation: 1 * oneEpochDuration(),
		MeshFailurePenaltyWeight:        meshWeight,
		MeshFailurePenaltyDecay:         scoreDecay(1 * oneEpochDuration()),
		InvalidMessageDeliveriesWeight:  -maxScore() / aggregateWeight,
		InvalidMessageDeliveriesDecay:   scoreDecay(50 * oneEpochDuration()),
	}
}

func defaultAggregateSubnetTopicParams(activeValidators uint64) *pubsub.TopicScoreParams {
	subnetCount := params.BeaconNetworkConfig().AttestationSubnetCount
	// Get weight for each specific subnet.
	topicWeight := attestationTotalWeight / float64(subnetCount)
	subnetWeight := activeValidators / subnetCount
	// Determine the amount of validators expected in a subnet in a single slot.
	numPerSlot := time.Duration(subnetWeight / uint64(params.BeaconConfig().SlotsPerEpoch))
	comsPerSlot := committeeCountPerSlot(activeValidators)
	exceedsThreshold := comsPerSlot >= 2*subnetCount/uint64(params.BeaconConfig().SlotsPerEpoch)
	firstDecay := time.Duration(1)
	meshDecay := time.Duration(4)
	if exceedsThreshold {
		firstDecay = 4
		meshDecay = 16
	}
	firstMessageCap := decay(scoreDecay(firstDecay*oneEpochDuration()), float64(numPerSlot*2/gossipSubD))
	firstMessageWeight := maxFirstDeliveryScore / firstMessageCap

	meshThreshold := decayThreshold(scoreDecay(meshDecay*oneEpochDuration()), float64(numPerSlot)/dampeningFactor)
	meshWeight := -maxScore() / (aggregateWeight * meshThreshold * meshThreshold)
	meshCap := 4 * meshThreshold
	return &pubsub.TopicScoreParams{
		TopicWeight:                     topicWeight,
		TimeInMeshWeight:                maxInMeshScore / inMeshCap(),
		TimeInMeshQuantum:               numPerSlot,
		TimeInMeshCap:                   inMeshCap(),
		FirstMessageDeliveriesWeight:    firstMessageWeight,
		FirstMessageDeliveriesDecay:     scoreDecay(firstDecay * oneEpochDuration()),
		FirstMessageDeliveriesCap:       firstMessageCap,
		MeshMessageDeliveriesWeight:     meshWeight,
		MeshMessageDeliveriesDecay:      scoreDecay(meshDecay * oneEpochDuration()),
		MeshMessageDeliveriesCap:        meshCap,
		MeshMessageDeliveriesThreshold:  meshThreshold,
		MeshMessageDeliveriesWindow:     2 * time.Second,
		MeshMessageDeliveriesActivation: 1 * oneEpochDuration(),
		MeshFailurePenaltyWeight:        meshWeight,
		MeshFailurePenaltyDecay:         scoreDecay(meshDecay * oneEpochDuration()),
		InvalidMessageDeliveriesWeight:  -maxScore() / float64(attestationTotalWeight/subnetCount),
		InvalidMessageDeliveriesDecay:   scoreDecay(50 * oneEpochDuration()),
	}
}

func defaultAttesterSlashingTopicParams() *pubsub.TopicScoreParams {
	return &pubsub.TopicScoreParams{
		TopicWeight:                     attesterSlashingWeight,
		TimeInMeshWeight:                maxInMeshScore / inMeshCap(),
		TimeInMeshQuantum:               inMeshTime(),
		TimeInMeshCap:                   inMeshCap(),
		FirstMessageDeliveriesWeight:    36,
		FirstMessageDeliveriesDecay:     scoreDecay(100 * oneEpochDuration()),
		FirstMessageDeliveriesCap:       1,
		MeshMessageDeliveriesWeight:     0,
		MeshMessageDeliveriesDecay:      0,
		MeshMessageDeliveriesCap:        0,
		MeshMessageDeliveriesThreshold:  0,
		MeshMessageDeliveriesWindow:     0,
		MeshMessageDeliveriesActivation: 0,
		MeshFailurePenaltyWeight:        0,
		MeshFailurePenaltyDecay:         0,
		InvalidMessageDeliveriesWeight:  -2000,
		InvalidMessageDeliveriesDecay:   scoreDecay(50 * oneEpochDuration()),
	}
}

func defaultProposerSlashingTopicParams() *pubsub.TopicScoreParams {
	return &pubsub.TopicScoreParams{
		TopicWeight:                     proposerSlashingWeight,
		TimeInMeshWeight:                maxInMeshScore / inMeshCap(),
		TimeInMeshQuantum:               inMeshTime(),
		TimeInMeshCap:                   inMeshCap(),
		FirstMessageDeliveriesWeight:    36,
		FirstMessageDeliveriesDecay:     scoreDecay(100 * oneEpochDuration()),
		FirstMessageDeliveriesCap:       1,
		MeshMessageDeliveriesWeight:     0,
		MeshMessageDeliveriesDecay:      0,
		MeshMessageDeliveriesCap:        0,
		MeshMessageDeliveriesThreshold:  0,
		MeshMessageDeliveriesWindow:     0,
		MeshMessageDeliveriesActivation: 0,
		MeshFailurePenaltyWeight:        0,
		MeshFailurePenaltyDecay:         0,
		InvalidMessageDeliveriesWeight:  -2000,
		InvalidMessageDeliveriesDecay:   scoreDecay(50 * oneEpochDuration()),
	}
}

func defaultVoluntaryExitTopicParams() *pubsub.TopicScoreParams {
	return &pubsub.TopicScoreParams{
		TopicWeight:                     voluntaryExitWeight,
		TimeInMeshWeight:                maxInMeshScore / inMeshCap(),
		TimeInMeshQuantum:               inMeshTime(),
		TimeInMeshCap:                   inMeshCap(),
		FirstMessageDeliveriesWeight:    2,
		FirstMessageDeliveriesDecay:     scoreDecay(100 * oneEpochDuration()),
		FirstMessageDeliveriesCap:       5,
		MeshMessageDeliveriesWeight:     0,
		MeshMessageDeliveriesDecay:      0,
		MeshMessageDeliveriesCap:        0,
		MeshMessageDeliveriesThreshold:  0,
		MeshMessageDeliveriesWindow:     0,
		MeshMessageDeliveriesActivation: 0,
		MeshFailurePenaltyWeight:        0,
		MeshFailurePenaltyDecay:         0,
		InvalidMessageDeliveriesWeight:  -2000,
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

func decayThreshold(decayRate, rate float64) float64 {
	return decay(decayRate, rate) * decayRate
}

func decay(decayRate, rate float64) float64 {
	return rate/1 - decayRate
}

func committeeCountPerSlot(activeValidators uint64) uint64 {
	// Use a static parameter for now rather than a dynamic one, we can use
	// the actual parameter later when we have figured out how to fix a circular
	// dependency in service startup order.
	return helpers.SlotCommitteeCount(activeValidators)
}

// Uses a very rough gauge for total aggregator size per slot.
func aggregatorsPerSlot(activeValidators uint64) uint64 {
	comms := committeeCountPerSlot(activeValidators)
	totalAggs := comms * params.BeaconConfig().TargetAggregatorsPerCommittee
	return totalAggs
}

// maxScore attainable by a peer.
func maxScore() float64 {
	totalWeight := beaconBlockWeight + aggregateWeight + attestationTotalWeight +
		attesterSlashingWeight + proposerSlashingWeight + voluntaryExitWeight
	return (maxInMeshScore + maxFirstDeliveryScore) * totalWeight
}

func inMeshTime() time.Duration {
	return 1 * oneSlotDuration()
}

func inMeshCap() float64 {
	return float64((3600 * time.Second) / inMeshTime())
}

func logGossipParameters(topic string, params *pubsub.TopicScoreParams) {
	// Exit early in the event the parameter struct is nil.
	if params == nil {
		return
	}
	rawParams := reflect.ValueOf(params).Elem()
	numOfFields := rawParams.NumField()

	fields := make(logrus.Fields, numOfFields)
	for i := 0; i < numOfFields; i++ {
		fields[reflect.TypeOf(params).Elem().Field(i).Name] = rawParams.Field(i).Interface()
	}
	log.WithFields(fields).Debugf("Topic Parameters for %s", topic)
}
