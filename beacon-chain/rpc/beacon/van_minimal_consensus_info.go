package beacon

import (
	"context"
	"encoding/hex"
	"fmt"
	types2 "github.com/gogo/protobuf/types"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sort"
)

// FutureEpochProposerList retrieves the validator assignments for future epoch (n + 1)
func (bs *Server) FutureEpochProposerList(
	ctx context.Context,
) (minimalConsensusInfo *ethpb.MinimalConsensusInfo, err error) {
	currentSlot := bs.GenesisTimeFetcher.CurrentSlot()
	// Add logic for epoch + 1
	recentState, err := bs.StateGen.StateBySlot(ctx, currentSlot)

	if nil != err {
		return
	}

	nextEpoch := helpers.NextEpoch(recentState)
	futureEpochState := recentState.Copy()
	futureEpochSlotStart, err := helpers.StartSlot(nextEpoch)

	if nil != err {
		return
	}

	err = futureEpochState.SetSlot(futureEpochSlotStart)

	if nil != err {
		return
	}

	_, proposerToSlot, err := helpers.CommitteeAssignments(futureEpochState, nextEpoch)

	if nil != err {
		return
	}

	startSlot, err := helpers.StartSlot(nextEpoch)

	if nil != err {
		return
	}

	endSlot, err := helpers.EndSlot(nextEpoch)

	if nil != err {
		return
	}

	publicKeyList := make([]string, 0)
	slotToPubKey := make(map[types.Slot]string)
	sortedSlotSlice := make([]float64, 0)

	for validatorIndex, slots := range proposerToSlot {
		pubKey := recentState.PubkeyAtIndex(validatorIndex)
		pubKeyString := fmt.Sprintf("0x%s", hex.EncodeToString(pubKey[:]))

		for _, slot := range slots {
			if slot >= startSlot && slot <= endSlot {
				slotToPubKey[slot] = pubKeyString
				sortedSlotSlice = append(sortedSlotSlice, float64(slot))
			}
		}
	}

	sort.Float64s(sortedSlotSlice)

	for _, slot := range sortedSlotSlice {
		publicKeyList = append(publicKeyList, slotToPubKey[types.Slot(slot)])
	}

	// It should never reach epoch 0 so no fallback on slot0 needed
	if len(publicKeyList) != int(params.BeaconConfig().SlotsPerEpoch) {
		err = fmt.Errorf(
			"invalid prpopser list len for epoch: %d, wanted: %d, got: %d",
			nextEpoch,
			int(params.BeaconConfig().SlotsPerEpoch),
			len(publicKeyList),
		)

		// Do not return any if they are invalid
		publicKeyList = make([]string, 0)
	}

	genesisTimeFetcher := bs.GenesisTimeFetcher
	genesisTime := genesisTimeFetcher.GenesisTime()

	futureEpochSlotStartTime, err := helpers.SlotToTime(uint64(genesisTime.Unix()), futureEpochSlotStart)

	if nil != err {
		return
	}

	minimalConsensusInfo = &ethpb.MinimalConsensusInfo{
		Epoch:            nextEpoch,
		ValidatorList:    publicKeyList,
		EpochTimeStart:   uint64(futureEpochSlotStartTime.Unix()),
		SlotTimeDuration: &types2.Duration{Seconds: int64(params.BeaconConfig().SecondsPerSlot)},
	}

	return
}

// MinimalConsensusInfoRange allows to subscribe into feed about minimalConsensusInformation from particular epoch
// TODO: Serve it in chunks, if recent epoch will be a very high number flood of responses could kill the connection
func (bs *Server) MinimalConsensusInfoRange(
	ctx context.Context,
	fromEpoch types.Epoch,
) (consensusInfos []*ethpb.MinimalConsensusInfo, err error) {
	consensusInfo, err := bs.MinimalConsensusInfo(ctx, fromEpoch)
	if nil != err {
		log.WithField("currentEpoch", "unknown").
			WithField("requestedEpoch", fromEpoch).Error(err.Error())

		return nil, err
	}

	consensusInfos = make([]*ethpb.MinimalConsensusInfo, 0)
	consensusInfos = append(consensusInfos, consensusInfo)

	currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())

	for tempEpochIndex := consensusInfo.Epoch; tempEpochIndex <= currentEpoch + 1; tempEpochIndex++ {
		minimalConsensusInfo, currentErr := bs.MinimalConsensusInfo(ctx, tempEpochIndex)
		if nil != currentErr {
			log.WithField("tempEpochIndex", tempEpochIndex).
				WithField("currentEpoch", currentEpoch).
				WithField("context", "epochNotFound").
				WithField("requestedEpoch", fromEpoch).Error(currentErr.Error())

			break
		}

		consensusInfos = append(consensusInfos, minimalConsensusInfo)
	}

	log.WithField("currentEpoch", currentEpoch).
		WithField("gathered", len(consensusInfos)).
		WithField("requestedEpoch", fromEpoch).Info("I should send epoch list")

	return
}

// MinimalConsensusInfo will give simple information about particular epoch
// If epoch is not present it will return an error
func (bs *Server) MinimalConsensusInfo(
	ctx context.Context,
	curEpoch types.Epoch,
) (minConsensusInfo *ethpb.MinimalConsensusInfo, err error) {
	log.WithField("prefix", "MinimalConsensusInfo")

	assignments, err := bs.getProposerListForEpoch(curEpoch)
	if nil != err {
		log.Errorf("[VAN_SUB] getProposerListForEpoch err = %s", err.Error())
		return nil, err
	}

	assignmentsSlice := make([]string, 0)
	slotToPubKey := make(map[types.Slot]string)
	sortedSlotSlice := make([]float64, 0)

	// Slot 0 was never signed by anybody
	if 0 == curEpoch {
		publicKeyBytes := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		currentString := fmt.Sprintf("0x%s", hex.EncodeToString(publicKeyBytes))
		assignmentsSlice = append(assignmentsSlice, currentString)
		slotToPubKey[0] = currentString
	}

	for _, assignment := range assignments.Assignments {
		for _, slot := range assignment.ProposerSlots {
			pubKeyString := fmt.Sprintf("0x%s", hex.EncodeToString(assignment.PublicKey))
			slotToPubKey[slot] = pubKeyString
			sortedSlotSlice = append(sortedSlotSlice, float64(slot))
		}
	}

	sort.Float64s(sortedSlotSlice)

	for _, slot := range sortedSlotSlice {
		assignmentsSlice = append(assignmentsSlice, slotToPubKey[types.Slot(slot)])
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

	genesisTime := bs.GenesisTimeFetcher.GenesisTime()
	startSlot, err := helpers.StartSlot(curEpoch)
	if nil != err {
		log.Errorf("[VAN_SUB] StartSlot err = %s", err.Error())
		return nil, err
	}
	epochStartTime, err := helpers.SlotToTime(uint64(genesisTime.Unix()), startSlot)
	if nil != err {
		log.Errorf("[VAN_SUB] SlotToTime err = %s", err.Error())
		return nil, err
	}

	minConsensusInfo = &ethpb.MinimalConsensusInfo{
		Epoch:            curEpoch,
		ValidatorList:    assignmentsSlice,
		EpochTimeStart:   uint64(epochStartTime.Unix()),
		SlotTimeDuration: &types2.Duration{Seconds: int64(params.BeaconConfig().SecondsPerSlot)},
	}

	log.Infof("[VAN_SUB] currEpoch = %#v", uint64(curEpoch))

	return minConsensusInfo, nil
}

func (bs *Server) getProposerListForEpoch(
	requestedEpoch types.Epoch,
) (*ethpb.ValidatorAssignments, error) {
	var (
		res         []*ethpb.ValidatorAssignments_CommitteeAssignment
		latestState iface.BeaconState
	)
	startSlot, err := helpers.StartSlot(requestedEpoch)

	if err != nil {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve startSlot for epoch %d: %v", requestedEpoch, err)
	}

	endSlot, err := helpers.EndSlot(requestedEpoch)

	if nil != err {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve endSlot for epoch %d: %v", requestedEpoch, err)
	}

	states, err := bs.BeaconDB.VanHighestSlotStatesBelow(bs.Ctx, endSlot)

	if nil != bs.Ctx.Err() {
		log.Infof("[VAN_SUB] getProposerListForEpoch bs.ctx err = %s", bs.Ctx.Err().Error())
	}

	if err != nil {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve archived state for epoch %d: %v", requestedEpoch, err)
	}

	statesCount := len(states)
	log.Debugf("[VAN_SUB] HighestSlotStatesBelow states len = %v", statesCount)

	if statesCount < 1 {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve any state by HighestSlotStatesBelow for endSlot %v", endSlot)
	}

	latestState = states[0]

	if statesCount > 1 {
		// Any state should return same proposer assignments so I pick first in slice
		for _, currentState := range states {
			log.Debugf("[VAN_SUB] Iterating over states, currentState.Slot = %v, startSlot = %v, endSlot = %v", currentState.Slot(), startSlot, endSlot)
			if currentState.Slot() >= startSlot && currentState.Slot() <= endSlot {
				latestState = currentState
				log.Debugf("[VAN_SUB] Iterating over states, currentState = %v, latestState = %v", currentState, latestState)

				break
			}
		}
	}

	if nil == latestState {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve any state for epoch %d", requestedEpoch)
	}

	// Initialize all committee related data.
	res, err = helpers.ProposerAssignments(latestState, requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}

	maxValidators := params.BeaconConfig().SlotsPerEpoch

	// We omit the genesis slot
	if requestedEpoch == 0 {
		maxValidators = maxValidators - 1
	}

	if len(res) != int(maxValidators) {
		return nil, fmt.Errorf("invalid validators len, expected: %d, got: %d, epoch: %#v", maxValidators, len(res), requestedEpoch)
	}

	return &ethpb.ValidatorAssignments{
		Epoch:       requestedEpoch,
		Assignments: res,
	}, nil
}

// StreamMinimalConsensusInfo to orchestrator client every single time an unconfirmed block is received by the beacon node.
func (bs *Server) StreamMinimalConsensusInfo(
	req *ethpb.MinimalConsensusInfoRequest,
	stream ethpb.BeaconChain_StreamMinimalConsensusInfoServer,
) error {
	if bs.SyncChecker.Syncing() {
		return status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	// If we are post-genesis time, then set the current epoch to
	// the number epochs since the genesis time, otherwise 0 by default.
	genesisTime := bs.GenesisTimeFetcher.GenesisTime()
	if genesisTime.IsZero() {
		return status.Error(codes.Unavailable, "genesis time is not set")
	}

	requestedFromEpoch := req.FromEpoch

	currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
	if requestedFromEpoch > currentEpoch {
		return status.Errorf(
			codes.InvalidArgument,
			errEpoch,
			currentEpoch,
			requestedFromEpoch,
		)
	}

	minimalConsensusInfoRange, err := bs.MinimalConsensusInfoRange(bs.Ctx, requestedFromEpoch)
	if err != nil {
		return status.Errorf(codes.Unavailable, "Could not get MinimalConsensusInfoRange for a stream: %v", err)
	}

	for _, minimalConsensusInfo := range minimalConsensusInfoRange {
		if err := stream.Send(minimalConsensusInfo); err != nil {
			return status.Errorf(codes.Unavailable, "Could not send minimalConsensusInfo over stream: %v", err)
		}
	}

	secondsPerEpoch := params.BeaconConfig().SecondsPerSlot * uint64(params.BeaconConfig().SlotsPerEpoch)
	epochTicker := slotutil.NewSlotTicker(bs.GenesisTimeFetcher.GenesisTime(), secondsPerEpoch)

	for {
		select {
		case <-epochTicker.C():
			res, err := bs.FutureEpochProposerList(bs.Ctx)
			if err != nil {
				return status.Errorf(codes.Unavailable, "Could not send minimalConsensusInfo over stream: %v", err)
			}
			if err := stream.Send(res); err != nil {
				return status.Errorf(codes.Internal, "Could not send minimalConsensusInfo response over stream: %v", err)
			}
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "[minimalConsensusInfo] Stream context canceled")
		case <-bs.Ctx.Done():
			return status.Error(codes.Canceled, "[minimalConsensusInfo] RPC context canceled")
		}
	}
}
