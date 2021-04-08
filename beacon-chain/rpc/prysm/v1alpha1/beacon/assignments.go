package beacon

import (
	"context"
	"encoding/hex"
	"fmt"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/subscriber/api/events"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"sort"
	"strconv"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const errEpoch = "Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d"

// ListValidatorAssignments retrieves the validator assignments for a given epoch,
// optional validator indices or public keys may be included to filter validator assignments.
func (bs *Server) ListValidatorAssignments(
	ctx context.Context, req *ethpb.ListValidatorAssignmentsRequest,
) (*ethpb.ValidatorAssignments, error) {
	if int(req.PageSize) > cmd.Get().MaxRPCPageSize {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Requested page size %d can not be greater than max size %d",
			req.PageSize,
			cmd.Get().MaxRPCPageSize,
		)
	}

	var res []*ethpb.ValidatorAssignments_CommitteeAssignment
	filtered := map[types.ValidatorIndex]bool{} // track filtered validators to prevent duplication in the response.
	filteredIndices := make([]types.ValidatorIndex, 0)
	var requestedEpoch types.Epoch
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListValidatorAssignmentsRequest_Genesis:
		if q.Genesis {
			requestedEpoch = 0
		}
	case *ethpb.ListValidatorAssignmentsRequest_Epoch:
		requestedEpoch = q.Epoch
	}

	currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
	if requestedEpoch > currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			errEpoch,
			currentEpoch,
			requestedEpoch,
		)
	}

	startSlot, err := helpers.StartSlot(requestedEpoch)
	if err != nil {
		return nil, err
	}
	requestedState, err := bs.StateGen.StateBySlot(ctx, startSlot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve archived state for epoch %d: %v", requestedEpoch, err)
	}

	// Filter out assignments by public keys.
	for _, pubKey := range req.PublicKeys {
		index, ok := requestedState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
		if !ok {
			return nil, status.Errorf(codes.NotFound, "Could not find validator index for public key %#x", pubKey)
		}
		filtered[index] = true
		filteredIndices = append(filteredIndices, index)
	}

	// Filter out assignments by validator indices.
	for _, index := range req.Indices {
		if !filtered[index] {
			filteredIndices = append(filteredIndices, index)
		}
	}

	activeIndices, err := helpers.ActiveValidatorIndices(requestedState, requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve active validator indices: %v", err)
	}
	if len(filteredIndices) == 0 {
		if len(activeIndices) == 0 {
			return &ethpb.ValidatorAssignments{
				Assignments:   make([]*ethpb.ValidatorAssignments_CommitteeAssignment, 0),
				TotalSize:     int32(0),
				NextPageToken: strconv.Itoa(0),
			}, nil
		}
		// If no filter was specified, return assignments from active validator indices with pagination.
		filteredIndices = activeIndices
	}

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), len(filteredIndices))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not paginate results: %v", err)
	}

	// Initialize all committee related data.
	committeeAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(requestedState, requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}

	for _, index := range filteredIndices[start:end] {
		if uint64(index) >= uint64(requestedState.NumValidators()) {
			return nil, status.Errorf(codes.OutOfRange, "Validator index %d >= validator count %d",
				index, requestedState.NumValidators())
		}
		comAssignment := committeeAssignments[index]
		pubkey := requestedState.PubkeyAtIndex(index)
		assign := &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: comAssignment.Committee,
			CommitteeIndex:   comAssignment.CommitteeIndex,
			AttesterSlot:     comAssignment.AttesterSlot,
			ProposerSlots:    proposerIndexToSlots[index],
			PublicKey:        pubkey[:],
			ValidatorIndex:   index,
		}
		res = append(res, assign)
	}

	return &ethpb.ValidatorAssignments{
		Epoch:         requestedEpoch,
		Assignments:   res,
		NextPageToken: nextPageToken,
		TotalSize:     int32(len(filteredIndices)),
	}, nil
}

// Deprecated: use GetMinimalConsensusInfoRange or FutureEpochProposerList
// TODO: remove this after protobufs will be updated for Server
func (bs *Server) NextEpochProposerList(
	ctx context.Context,
	empty *ptypes.Empty,
) (assignments *ethpb.ValidatorAssignments, err error) {
	return
}

// FutureEpochProposerList retrieves the validator assignments for future epoch (n + 1)
func (bs *Server) FutureEpochProposerList(
	ctx context.Context,
) (minimalConsensusInfo *events.MinimalEpochConsensusInfo, err error) {
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

	minimalConsensusInfo = &events.MinimalEpochConsensusInfo{
		Epoch:            uint64(nextEpoch),
		ValidatorList:    publicKeyList,
		EpochStartTime:   uint64(futureEpochSlotStartTime.Unix()),
		SlotTimeDuration: time.Duration(params.BeaconConfig().SecondsPerSlot),
	}

	return
}

// GetMinimalConsensusInfoRange allows to subscribe into feed about minimalConsensusInformation from particular epoch
// TODO: Serve it in chunks, if recent epoch will be a very high number flood of responses could kill the connection
func (bs *Server) GetMinimalConsensusInfoRange(
	ctx context.Context,
	fromEpoch types.Epoch,
) (consensusInfos []*events.MinimalEpochConsensusInfo, err error) {
	consensusInfo, err := bs.GetMinimalConsensusInfo(ctx, fromEpoch)

	if nil != err {
		log.WithField("currentEpoch", "unknown").
			WithField("requestedEpoch", fromEpoch).Error(err.Error())

		return nil, err
	}

	consensusInfos = make([]*events.MinimalEpochConsensusInfo, 0)
	consensusInfos = append(consensusInfos, consensusInfo)
	tempEpochIndex := consensusInfo.Epoch

	for {
		tempEpochIndex++
		minimalConsensusInfo, currentErr := bs.GetMinimalConsensusInfo(ctx, types.Epoch(tempEpochIndex))

		if nil != currentErr {
			log.WithField("currentEpoch", tempEpochIndex).
				WithField("context", "epochNotFound").
				WithField("requestedEpoch", fromEpoch).Error(currentErr.Error())

			break
		}

		consensusInfos = append(consensusInfos, minimalConsensusInfo)
	}

	log.WithField("currentEpoch", tempEpochIndex).
		WithField("gathered", len(consensusInfos)).
		WithField("requestedEpoch", fromEpoch).Info("I should send epoch list")

	return
}

// GetMinimalConsensusInfo will give simple information about particular epoch
// If epoch is not present it will return an error
func (bs *Server) GetMinimalConsensusInfo(
	ctx context.Context,
	curEpoch types.Epoch,
) (minConsensusInfo *events.MinimalEpochConsensusInfo, err error) {
	log.WithField("prefix", "GetMinimalConsensusInfo")

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

	minConsensusInfo = &events.MinimalEpochConsensusInfo{
		Epoch:            uint64(curEpoch),
		ValidatorList:    assignmentsSlice,
		EpochStartTime:   uint64(epochStartTime.Unix()),
		SlotTimeDuration: time.Duration(params.BeaconConfig().SecondsPerSlot),
	}

	log.Infof("[VAN_SUB] currEpoch = %#v", uint64(curEpoch))

	return minConsensusInfo, nil
}

func (bs *Server) getProposerListForEpoch(
	requestedEpoch types.Epoch,
) (*ethpb.ValidatorAssignments, error) {
	var (
		res         []*ethpb.ValidatorAssignments_CommitteeAssignment
		latestState *state.BeaconState
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

	states, err := bs.BeaconDB.HighestSlotStatesBelow(bs.Ctx, endSlot)

	if nil != bs.Ctx.Err() {
		log.Infof("[VAN_SUB] getProposerListForEpoch bs.ctx err = %s", bs.Ctx.Err().Error())
	}

	if err != nil {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve archived state for epoch %d: %v", requestedEpoch, err)
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
			codes.Internal, "Could not retrieve any state for epoch %d", requestedEpoch)
	}

	// Initialize all committee related data.
	proposerIndexToSlots, err := helpers.ProposerAssignments(latestState, requestedEpoch)
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
