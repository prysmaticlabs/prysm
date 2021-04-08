package blockchain

import (
	"encoding/hex"
	"fmt"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/subscriber/api/events"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

type MinimalEpochConsensusInfo struct {
	Epoch            uint64        `json:"epoch"`
	ValidatorList    []string      `json:"validatorList"`
	EpochStartTime   uint64        `json:"epochTimeStart"`
	SlotTimeDuration time.Duration `json:"slotTimeDuration"`
}

func (s *Service) MinimalConsensusInfo() (minConsensusInfo []*ethpb.MinimalEpochConsensusInfo, err error) {
	log.WithField("prefix", "GetPastMinimalConsensusInfo")

	currentEpoch := helpers.SlotToEpoch(s.CurrentSlot())
	assignments, err := s.getPastProposerListForEpoch(currentEpoch)
	if nil != err {
		log.Errorf("[VAN_SUB] getProposerListForEpoch err = %s", err.Error())
		return nil, err
	}

	assignmentsSlice := make([]string, 0)

	// Slot 0 was never signed by anybody
	if 0 == currentEpoch {
		publicKeyBytes := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		currentString := fmt.Sprintf("0x%s", hex.EncodeToString(publicKeyBytes))
		assignmentsSlice = append(assignmentsSlice, currentString)
	}

	for _, assigment := range assignments.Assignments {
		currentString := fmt.Sprintf("0x%s", hex.EncodeToString(assigment.PublicKey))
		assignmentsSlice = append(assignmentsSlice, currentString)
	}

	expectedValidators := int(params.BeaconConfig().SlotsPerEpoch)

	if len(assignmentsSlice) != expectedValidators {
		err := fmt.Errorf(
			"not enough assignments, expected: %d, got: %d",
			expectedValidators,
			len(assignmentsSlice),
		)
		log.Errorf("[VAN_SUB] Assignments err = %s", err.Error())

		return nil, err
	}

	genesisTime := s.genesisTime
	startSlot, err := helpers.StartSlot(currentEpoch)
	if nil != err {
		log.Errorf("[VAN_SUB] StartSlot err = %s", err.Error())
		return nil, err
	}
	epochStartTime, err := helpers.SlotToTime(uint64(genesisTime.Unix()), startSlot)
	if nil != err {
		log.Errorf("[VAN_SUB] SlotToTime err = %s", err.Error())
		return nil, err
	}

	minConsensusInfo = &events.MinimalEpochConsensusInfo{
		Epoch:            uint64(currentEpoch),
		ValidatorList:    assignmentsSlice,
		EpochStartTime:   uint64(epochStartTime.Unix()),
		SlotTimeDuration: time.Duration(params.BeaconConfig().SecondsPerSlot),
	}

	log.Infof("[VAN_SUB] currEpoch = %#v", uint64(toEpoch))

	return minConsensusInfo, nil
}

func (s *Service) getPastProposerListForEpoch(currentEpoch types.Epoch) (*ethpb.ValidatorAssignments, error) {
	var (
		res         []*ethpb.ValidatorAssignments_CommitteeAssignment
		latestState *state.BeaconState
	)

	startSlot, err := helpers.StartSlot(currentEpoch)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve startSlot for epoch %d: %v", currentEpoch, err)
	}

	endSlot, err := helpers.EndSlot(currentEpoch)
	if nil != err {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve endSlot for epoch %d: %v", currentEpoch, err)
	}

	states, err := s.beaconDB.HighestSlotStatesBelow(s.ctx, endSlot)
	if nil != s.ctx.Err() {
		log.Infof("[VAN_SUB] getProposerListForEpoch bs.ctx err = %s", s.ctx.Err().Error())
	}
	if err != nil {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve archived state for epoch %d: %v", currentEpoch, err)
	}

	log.Debugf("[VAN_SUB] HighestSlotStatesBelow states len = %v", len(states))

	// Any state should return same proposer assignments so I pick first in slice
	for _, currentState := range states {
		if currentState.Slot() >= startSlot && currentState.Slot() <= endSlot {
			latestState = currentState

			break
		}
	}

	if nil == latestState {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve any state for epoch %d", currentEpoch)
	}

	// Initialize all committee related data.
	proposerIndexToSlots, err := helpers.ProposerAssignments(latestState, currentEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}

	for index, proposerSlots := range proposerIndexToSlots {
		pubkey := latestState.PubkeyAtIndex(index)
		assign := &ethpb.ValidatorAssignments_CommitteeAssignment{
			ProposerSlots:  proposerSlots,
			PublicKey:      pubkey[:],
			ValidatorIndex: index,
		}
		res = append(res, assign)
	}

	maxValidators := params.BeaconConfig().SlotsPerEpoch

	// We omit the genesis slot
	if currentEpoch == 0 {
		maxValidators = maxValidators - 1
	}

	if len(res) != int(maxValidators) {
		return nil, fmt.Errorf("invalid validators len, expected: %d, got: %d, epoch: %#v", maxValidators, len(res), currentEpoch)
	}

	return &ethpb.ValidatorAssignments{
		Epoch:       currentEpoch,
		Assignments: res,
	}, nil
}
