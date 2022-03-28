package beacon

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/api/pagination"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	coreTime "github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ListValidatorBalances retrieves the validator balances for a given set of public keys.
// An optional Epoch parameter is provided to request historical validator balances from
// archived, persistent data.
func (bs *Server) ListValidatorBalances(
	ctx context.Context,
	req *ethpb.ListValidatorBalancesRequest,
) (*ethpb.ValidatorBalances, error) {
	if int(req.PageSize) > cmd.Get().MaxRPCPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, cmd.Get().MaxRPCPageSize)
	}

	if bs.GenesisTimeFetcher == nil {
		return nil, status.Errorf(codes.Internal, "Nil genesis time fetcher")
	}
	currentEpoch := slots.ToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
	requestedEpoch := currentEpoch
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListValidatorBalancesRequest_Epoch:
		requestedEpoch = q.Epoch
	case *ethpb.ListValidatorBalancesRequest_Genesis:
		requestedEpoch = 0
	}

	if requestedEpoch > currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			errEpoch,
			currentEpoch,
			requestedEpoch,
		)
	}
	res := make([]*ethpb.ValidatorBalances_Balance, 0)
	filtered := map[types.ValidatorIndex]bool{} // Track filtered validators to prevent duplication in the response.

	startSlot, err := slots.EpochStart(requestedEpoch)
	if err != nil {
		return nil, err
	}
	requestedState, err := bs.ReplayerBuilder.ReplayerForSlot(startSlot).ReplayBlocks(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("error replaying blocks for state at slot %d: %v", startSlot, err))
	}

	vals := requestedState.Validators()
	balances := requestedState.Balances()
	balancesCount := len(balances)
	for _, pubKey := range req.PublicKeys {
		// Skip empty public key.
		if len(pubKey) == 0 {
			continue
		}
		pubkeyBytes := bytesutil.ToBytes48(pubKey)
		index, ok := requestedState.ValidatorIndexByPubkey(pubkeyBytes)
		if !ok {
			// We continue the loop if one validator in the request is not found.
			res = append(res, &ethpb.ValidatorBalances_Balance{
				Status: "UNKNOWN",
			})
			balancesCount = len(res)
			continue
		}
		filtered[index] = true

		if uint64(index) >= uint64(len(balances)) {
			return nil, status.Errorf(codes.OutOfRange, "Validator index %d >= balance list %d",
				index, len(balances))
		}

		val := vals[index]
		st := validatorStatus(val, requestedEpoch)
		res = append(res, &ethpb.ValidatorBalances_Balance{
			PublicKey: pubKey,
			Index:     index,
			Balance:   balances[index],
			Status:    st.String(),
		})
		balancesCount = len(res)
	}

	for _, index := range req.Indices {
		if uint64(index) >= uint64(len(balances)) {
			return nil, status.Errorf(codes.OutOfRange, "Validator index %d >= balance list %d",
				index, len(balances))
		}

		if !filtered[index] {
			val := vals[index]
			st := validatorStatus(val, requestedEpoch)
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: vals[index].PublicKey,
				Index:     index,
				Balance:   balances[index],
				Status:    st.String(),
			})
		}
		balancesCount = len(res)
	}
	// Depending on the indices and public keys given, results might not be sorted.
	sort.Slice(res, func(i, j int) bool {
		return res[i].Index < res[j].Index
	})

	// If there are no balances, we simply return a response specifying this.
	// Otherwise, attempting to paginate 0 balances below would result in an error.
	if balancesCount == 0 {
		return &ethpb.ValidatorBalances{
			Epoch:         requestedEpoch,
			Balances:      make([]*ethpb.ValidatorBalances_Balance, 0),
			TotalSize:     int32(0),
			NextPageToken: strconv.Itoa(0),
		}, nil
	}

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), balancesCount)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Could not paginate results: %v",
			err,
		)
	}

	if len(req.Indices) == 0 && len(req.PublicKeys) == 0 {
		// Return everything.
		for i := start; i < end; i++ {
			pubkey := requestedState.PubkeyAtIndex(types.ValidatorIndex(i))
			val := vals[i]
			st := validatorStatus(val, requestedEpoch)
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: pubkey[:],
				Index:     types.ValidatorIndex(i),
				Balance:   balances[i],
				Status:    st.String(),
			})
		}
		return &ethpb.ValidatorBalances{
			Epoch:         requestedEpoch,
			Balances:      res,
			TotalSize:     int32(balancesCount),
			NextPageToken: nextPageToken,
		}, nil
	}

	if end > len(res) || end < start {
		return nil, status.Error(codes.OutOfRange, "Request exceeds response length")
	}

	return &ethpb.ValidatorBalances{
		Epoch:         requestedEpoch,
		Balances:      res[start:end],
		TotalSize:     int32(balancesCount),
		NextPageToken: nextPageToken,
	}, nil
}

// ListValidators retrieves the current list of active validators with an optional historical epoch flag to
// to retrieve validator set in time.
func (bs *Server) ListValidators(
	ctx context.Context,
	req *ethpb.ListValidatorsRequest,
) (*ethpb.Validators, error) {
	if int(req.PageSize) > cmd.Get().MaxRPCPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, cmd.Get().MaxRPCPageSize)
	}

	currentEpoch := slots.ToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
	requestedEpoch := currentEpoch

	switch q := req.QueryFilter.(type) {
	case *ethpb.ListValidatorsRequest_Genesis:
		if q.Genesis {
			requestedEpoch = 0
		}
	case *ethpb.ListValidatorsRequest_Epoch:
		if q.Epoch > currentEpoch {
			return nil, status.Errorf(
				codes.InvalidArgument,
				errEpoch,
				currentEpoch,
				q.Epoch,
			)
		}
		requestedEpoch = q.Epoch
	}
	var reqState state.BeaconState
	var err error
	if requestedEpoch != currentEpoch {
		var s types.Slot
		s, err = slots.EpochStart(requestedEpoch)
		if err != nil {
			return nil, err
		}
		reqState, err = bs.ReplayerBuilder.ReplayerForSlot(s).ReplayBlocks(ctx)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("error replaying blocks for state at slot %d: %v", s, err))
		}
	} else {
		reqState, err = bs.HeadFetcher.HeadState(ctx)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get requested state: %v", err)
	}
	if reqState == nil || reqState.IsNil() {
		return nil, status.Error(codes.Internal, "Requested state is nil")
	}

	s, err := slots.EpochStart(requestedEpoch)
	if err != nil {
		return nil, err
	}
	if s > reqState.Slot() {
		reqState = reqState.Copy()
		reqState, err = transition.ProcessSlots(ctx, reqState, s)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not process slots up to epoch %d: %v",
				requestedEpoch,
				err,
			)
		}
	}

	validatorList := make([]*ethpb.Validators_ValidatorContainer, 0)

	for _, index := range req.Indices {
		val, err := reqState.ValidatorAtIndex(index)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator: %v", err)
		}
		validatorList = append(validatorList, &ethpb.Validators_ValidatorContainer{
			Index:     index,
			Validator: val,
		})
	}

	for _, pubKey := range req.PublicKeys {
		// Skip empty public key.
		if len(pubKey) == 0 {
			continue
		}
		pubkeyBytes := bytesutil.ToBytes48(pubKey)
		index, ok := reqState.ValidatorIndexByPubkey(pubkeyBytes)
		if !ok {
			continue
		}
		val, err := reqState.ValidatorAtIndex(index)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator: %v", err)
		}
		validatorList = append(validatorList, &ethpb.Validators_ValidatorContainer{
			Index:     index,
			Validator: val,
		})
	}
	// Depending on the indices and public keys given, results might not be sorted.
	sort.Slice(validatorList, func(i, j int) bool {
		return validatorList[i].Index < validatorList[j].Index
	})

	if len(req.PublicKeys) == 0 && len(req.Indices) == 0 {
		for i := types.ValidatorIndex(0); uint64(i) < uint64(reqState.NumValidators()); i++ {
			val, err := reqState.ValidatorAtIndex(i)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get validator: %v", err)
			}
			validatorList = append(validatorList, &ethpb.Validators_ValidatorContainer{
				Index:     i,
				Validator: val,
			})
		}
	}

	// Filter active validators if the request specifies it.
	res := validatorList
	if req.Active {
		filteredValidators := make([]*ethpb.Validators_ValidatorContainer, 0)
		for _, item := range validatorList {
			if helpers.IsActiveValidator(item.Validator, requestedEpoch) {
				filteredValidators = append(filteredValidators, item)
			}
		}
		res = filteredValidators
	}

	validatorCount := len(res)
	// If there are no items, we simply return a response specifying this.
	// Otherwise, attempting to paginate 0 validators below would result in an error.
	if validatorCount == 0 {
		return &ethpb.Validators{
			ValidatorList: make([]*ethpb.Validators_ValidatorContainer, 0),
			TotalSize:     int32(0),
			NextPageToken: strconv.Itoa(0),
		}, nil
	}

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), validatorCount)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Could not paginate results: %v",
			err,
		)
	}

	return &ethpb.Validators{
		ValidatorList: res[start:end],
		TotalSize:     int32(validatorCount),
		NextPageToken: nextPageToken,
	}, nil
}

// GetValidator information from any validator in the registry by index or public key.
func (bs *Server) GetValidator(
	ctx context.Context, req *ethpb.GetValidatorRequest,
) (*ethpb.Validator, error) {
	var requestingIndex bool
	var index types.ValidatorIndex
	var pubKey []byte
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorRequest_Index:
		index = q.Index
		requestingIndex = true
	case *ethpb.GetValidatorRequest_PublicKey:
		pubKey = q.PublicKey
	default:
		return nil, status.Error(
			codes.InvalidArgument,
			"Need to specify either validator index or public key in request",
		)
	}
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	if requestingIndex {
		if uint64(index) >= uint64(headState.NumValidators()) {
			return nil, status.Errorf(
				codes.OutOfRange,
				"Requesting index %d, but there are only %d validators",
				index,
				headState.NumValidators(),
			)
		}
		return headState.ValidatorAtIndex(index)
	}
	pk48 := bytesutil.ToBytes48(pubKey)
	for i := types.ValidatorIndex(0); uint64(i) < uint64(headState.NumValidators()); i++ {
		keyFromState := headState.PubkeyAtIndex(i)
		if keyFromState == pk48 {
			return headState.ValidatorAtIndex(i)
		}
	}
	return nil, status.Error(codes.NotFound, "No validator matched filter criteria")
}

// GetValidatorActiveSetChanges retrieves the active set changes for a given epoch.
//
// This data includes any activations, voluntary exits, and involuntary
// ejections.
func (bs *Server) GetValidatorActiveSetChanges(
	ctx context.Context, req *ethpb.GetValidatorActiveSetChangesRequest,
) (*ethpb.ActiveSetChanges, error) {
	currentEpoch := slots.ToEpoch(bs.GenesisTimeFetcher.CurrentSlot())

	var requestedEpoch types.Epoch
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorActiveSetChangesRequest_Genesis:
		requestedEpoch = 0
	case *ethpb.GetValidatorActiveSetChangesRequest_Epoch:
		requestedEpoch = q.Epoch
	default:
		requestedEpoch = currentEpoch
	}
	if requestedEpoch > currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			errEpoch,
			currentEpoch,
			requestedEpoch,
		)
	}

	s, err := slots.EpochStart(requestedEpoch)
	if err != nil {
		return nil, err
	}
	requestedState, err := bs.ReplayerBuilder.ReplayerForSlot(s).ReplayBlocks(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("error replaying blocks for state at slot %d: %v", s, err))
	}

	activeValidatorCount, err := helpers.ActiveValidatorCount(ctx, requestedState, coreTime.CurrentEpoch(requestedState))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
	}
	vs := requestedState.Validators()
	activatedIndices := validators.ActivatedValidatorIndices(coreTime.CurrentEpoch(requestedState), vs)
	exitedIndices, err := validators.ExitedValidatorIndices(coreTime.CurrentEpoch(requestedState), vs, activeValidatorCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine exited validator indices: %v", err)
	}
	slashedIndices := validators.SlashedValidatorIndices(coreTime.CurrentEpoch(requestedState), vs)
	ejectedIndices, err := validators.EjectedValidatorIndices(coreTime.CurrentEpoch(requestedState), vs, activeValidatorCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine ejected validator indices: %v", err)
	}

	// Retrieve public keys for the indices.
	activatedKeys := make([][]byte, len(activatedIndices))
	exitedKeys := make([][]byte, len(exitedIndices))
	slashedKeys := make([][]byte, len(slashedIndices))
	ejectedKeys := make([][]byte, len(ejectedIndices))
	for i, idx := range activatedIndices {
		pubkey := requestedState.PubkeyAtIndex(idx)
		activatedKeys[i] = pubkey[:]
	}
	for i, idx := range exitedIndices {
		pubkey := requestedState.PubkeyAtIndex(idx)
		exitedKeys[i] = pubkey[:]
	}
	for i, idx := range slashedIndices {
		pubkey := requestedState.PubkeyAtIndex(idx)
		slashedKeys[i] = pubkey[:]
	}
	for i, idx := range ejectedIndices {
		pubkey := requestedState.PubkeyAtIndex(idx)
		ejectedKeys[i] = pubkey[:]
	}
	return &ethpb.ActiveSetChanges{
		Epoch:               requestedEpoch,
		ActivatedPublicKeys: activatedKeys,
		ActivatedIndices:    activatedIndices,
		ExitedPublicKeys:    exitedKeys,
		ExitedIndices:       exitedIndices,
		SlashedPublicKeys:   slashedKeys,
		SlashedIndices:      slashedIndices,
		EjectedPublicKeys:   ejectedKeys,
		EjectedIndices:      ejectedIndices,
	}, nil
}

// GetValidatorParticipation retrieves the validator participation information for a given epoch,
// it returns the information about validator's participation rate in voting on the proof of stake
// rules based on their balance compared to the total active validator balance.
func (bs *Server) GetValidatorParticipation(
	ctx context.Context, req *ethpb.GetValidatorParticipationRequest,
) (*ethpb.ValidatorParticipationResponse, error) {
	currentSlot := bs.GenesisTimeFetcher.CurrentSlot()
	currentEpoch := slots.ToEpoch(currentSlot)

	var requestedEpoch types.Epoch
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorParticipationRequest_Genesis:
		requestedEpoch = 0
	case *ethpb.GetValidatorParticipationRequest_Epoch:
		requestedEpoch = q.Epoch
	default:
		requestedEpoch = currentEpoch
	}

	if requestedEpoch > currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch greater than current epoch, current epoch %d, requesting %d",
			currentEpoch,
			requestedEpoch,
		)
	}
	// Use the last slot of requested epoch to obtain current and previous epoch attestations.
	// This ensures that we don't miss previous attestations when input requested epochs.
	startSlot, err := slots.EpochEnd(requestedEpoch)
	if err != nil {
		return nil, err
	}
	// Get as close as we can to the end of the current epoch without going past the curent slot.
	// The above check ensures a future *epoch* isn't requested, but the end slot of the requested epoch could still
	// be past the current slot. In that case, use the current slot as the best approximation of the requested epoch.
	// Replayer will make sure the slot ultimately used is canonical.
	if startSlot > currentSlot {
		startSlot = currentSlot
	}

	// ReplayerBuilder ensures that a canonical chain is followed to the slot
	beaconState, err := bs.ReplayerBuilder.ReplayerForSlot(startSlot).ReplayBlocks(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("error replaying blocks for state at slot %d: %v", startSlot, err))
	}
	var v []*precompute.Validator
	var b *precompute.Balance
	switch beaconState.Version() {
	case version.Phase0:
		v, b, err = precompute.New(ctx, beaconState)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not set up pre compute instance: %v", err)
		}
		_, b, err = precompute.ProcessAttestations(ctx, beaconState, v, b)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not pre compute attestations: %v", err)
		}
	case version.Altair, version.Bellatrix:
		v, b, err = altair.InitializePrecomputeValidators(ctx, beaconState)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not set up altair pre compute instance: %v", err)
		}
		_, b, err = altair.ProcessEpochParticipation(ctx, beaconState, b, v)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not pre compute attestations: %v", err)
		}
	default:
		return nil, status.Errorf(codes.Internal, "Invalid state type retrieved with a version of %d", beaconState.Version())
	}

	p := &ethpb.ValidatorParticipationResponse{
		Epoch:     requestedEpoch,
		Finalized: requestedEpoch <= bs.FinalizationFetcher.FinalizedCheckpt().Epoch,
		Participation: &ethpb.ValidatorParticipation{
			// TODO(7130): Remove these three deprecated fields.
			GlobalParticipationRate:          float32(b.PrevEpochTargetAttested) / float32(b.ActivePrevEpoch),
			VotedEther:                       b.PrevEpochTargetAttested,
			EligibleEther:                    b.ActivePrevEpoch,
			CurrentEpochActiveGwei:           b.ActiveCurrentEpoch,
			CurrentEpochAttestingGwei:        b.CurrentEpochAttested,
			CurrentEpochTargetAttestingGwei:  b.CurrentEpochTargetAttested,
			PreviousEpochActiveGwei:          b.ActivePrevEpoch,
			PreviousEpochAttestingGwei:       b.PrevEpochAttested,
			PreviousEpochTargetAttestingGwei: b.PrevEpochTargetAttested,
			PreviousEpochHeadAttestingGwei:   b.PrevEpochHeadAttested,
		},
	}

	return p, nil
}

// GetValidatorQueue retrieves the current validator queue information.
func (bs *Server) GetValidatorQueue(
	ctx context.Context, _ *emptypb.Empty,
) (*ethpb.ValidatorQueue, error) {
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	// Queue the validators whose eligible to activate and sort them by activation eligibility epoch number.
	// Additionally, determine those validators queued to exit
	awaitingExit := make([]types.ValidatorIndex, 0)
	exitEpochs := make([]types.Epoch, 0)
	activationQ := make([]types.ValidatorIndex, 0)
	vals := headState.Validators()
	for idx, validator := range vals {
		eligibleActivated := validator.ActivationEligibilityEpoch != params.BeaconConfig().FarFutureEpoch
		canBeActive := validator.ActivationEpoch >= helpers.ActivationExitEpoch(headState.FinalizedCheckpointEpoch())
		if eligibleActivated && canBeActive {
			activationQ = append(activationQ, types.ValidatorIndex(idx))
		}
		if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochs = append(exitEpochs, validator.ExitEpoch)
			awaitingExit = append(awaitingExit, types.ValidatorIndex(idx))
		}
	}
	sort.Slice(activationQ, func(i, j int) bool {
		return vals[i].ActivationEligibilityEpoch < vals[j].ActivationEligibilityEpoch
	})
	sort.Slice(awaitingExit, func(i, j int) bool {
		return vals[i].WithdrawableEpoch < vals[j].WithdrawableEpoch
	})

	// Only activate just enough validators according to the activation churn limit.
	activeValidatorCount, err := helpers.ActiveValidatorCount(ctx, headState, coreTime.CurrentEpoch(headState))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
	}
	churnLimit, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute churn limit: %v", err)
	}

	exitQueueEpoch := types.Epoch(0)
	for _, i := range exitEpochs {
		if exitQueueEpoch < i {
			exitQueueEpoch = i
		}
	}
	exitQueueChurn := uint64(0)
	for _, val := range vals {
		if val.ExitEpoch == exitQueueEpoch {
			exitQueueChurn++
		}
	}
	// Prevent churn limit from causing index out of bound issues.
	if churnLimit < exitQueueChurn {
		// If we are above the churn limit, we simply increase the churn by one.
		exitQueueEpoch++
	}

	// We use the exit queue churn to determine if we have passed a churn limit.
	minEpoch := exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	exitQueueIndices := make([]types.ValidatorIndex, 0)
	for _, valIdx := range awaitingExit {
		val := vals[valIdx]
		// Ensure the validator has not yet exited before adding its index to the exit queue.
		if val.WithdrawableEpoch < minEpoch && !validatorHasExited(val, coreTime.CurrentEpoch(headState)) {
			exitQueueIndices = append(exitQueueIndices, valIdx)
		}
	}

	// Get the public keys for the validators in the queues up to the allowed churn limits.
	activationQueueKeys := make([][]byte, len(activationQ))
	exitQueueKeys := make([][]byte, len(exitQueueIndices))
	for i, idx := range activationQ {
		activationQueueKeys[i] = vals[idx].PublicKey
	}
	for i, idx := range exitQueueIndices {
		exitQueueKeys[i] = vals[idx].PublicKey
	}

	return &ethpb.ValidatorQueue{
		ChurnLimit:                 churnLimit,
		ActivationPublicKeys:       activationQueueKeys,
		ExitPublicKeys:             exitQueueKeys,
		ActivationValidatorIndices: activationQ,
		ExitValidatorIndices:       exitQueueIndices,
	}, nil
}

// GetValidatorPerformance reports the validator's latest balance along with other important metrics on
// rewards and penalties throughout its lifecycle in the beacon chain.
func (bs *Server) GetValidatorPerformance(
	ctx context.Context, req *ethpb.ValidatorPerformanceRequest,
) (*ethpb.ValidatorPerformanceResponse, error) {
	if bs.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	currSlot := bs.GenesisTimeFetcher.CurrentSlot()

	if currSlot > headState.Slot() {
		headRoot, err := bs.HeadFetcher.HeadRoot(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve head root: %v", err)
		}
		headState, err = transition.ProcessSlotsUsingNextSlotCache(ctx, headState, headRoot, currSlot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", currSlot, err)
		}
	}
	var validatorSummary []*precompute.Validator
	switch headState.Version() {
	case version.Phase0:
		vp, bp, err := precompute.New(ctx, headState)
		if err != nil {
			return nil, err
		}
		vp, bp, err = precompute.ProcessAttestations(ctx, headState, vp, bp)
		if err != nil {
			return nil, err
		}
		headState, err = precompute.ProcessRewardsAndPenaltiesPrecompute(headState, bp, vp, precompute.AttestationsDelta, precompute.ProposersDelta)
		if err != nil {
			return nil, err
		}
		validatorSummary = vp
	case version.Altair:
		vp, bp, err := altair.InitializePrecomputeValidators(ctx, headState)
		if err != nil {
			return nil, err
		}
		vp, bp, err = altair.ProcessEpochParticipation(ctx, headState, bp, vp)
		if err != nil {
			return nil, err
		}
		headState, vp, err = altair.ProcessInactivityScores(ctx, headState, vp)
		if err != nil {
			return nil, err
		}
		headState, err = altair.ProcessRewardsAndPenaltiesPrecompute(headState, bp, vp)
		if err != nil {
			return nil, err
		}
		validatorSummary = vp
	}

	responseCap := len(req.Indices) + len(req.PublicKeys)
	validatorIndices := make([]types.ValidatorIndex, 0, responseCap)
	missingValidators := make([][]byte, 0, responseCap)

	filtered := map[types.ValidatorIndex]bool{} // Track filtered validators to prevent duplication in the response.
	// Convert the list of validator public keys to validator indices and add to the indices set.
	for _, pubKey := range req.PublicKeys {
		// Skip empty public key.
		if len(pubKey) == 0 {
			continue
		}
		pubkeyBytes := bytesutil.ToBytes48(pubKey)
		idx, ok := headState.ValidatorIndexByPubkey(pubkeyBytes)
		if !ok {
			// Validator index not found, track as missing.
			missingValidators = append(missingValidators, pubKey)
			continue
		}
		if !filtered[idx] {
			validatorIndices = append(validatorIndices, idx)
			filtered[idx] = true
		}
	}
	// Add provided indices to the indices set.
	for _, idx := range req.Indices {
		if !filtered[idx] {
			validatorIndices = append(validatorIndices, idx)
			filtered[idx] = true
		}
	}
	// Depending on the indices and public keys given, results might not be sorted.
	sort.Slice(validatorIndices, func(i, j int) bool {
		return validatorIndices[i] < validatorIndices[j]
	})

	currentEpoch := coreTime.CurrentEpoch(headState)
	responseCap = len(validatorIndices)
	pubKeys := make([][]byte, 0, responseCap)
	beforeTransitionBalances := make([]uint64, 0, responseCap)
	afterTransitionBalances := make([]uint64, 0, responseCap)
	effectiveBalances := make([]uint64, 0, responseCap)
	inclusionSlots := make([]types.Slot, 0, responseCap)
	inclusionDistances := make([]types.Slot, 0, responseCap)
	correctlyVotedSource := make([]bool, 0, responseCap)
	correctlyVotedTarget := make([]bool, 0, responseCap)
	correctlyVotedHead := make([]bool, 0, responseCap)
	inactivityScores := make([]uint64, 0, responseCap)
	// Append performance summaries.
	// Also track missing validators using public keys.
	for _, idx := range validatorIndices {
		val, err := headState.ValidatorAtIndexReadOnly(idx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not get validator: %v", err)
		}
		pubKey := val.PublicKey()
		if uint64(idx) >= uint64(len(validatorSummary)) {
			// Not listed in validator summary yet; treat it as missing.
			missingValidators = append(missingValidators, pubKey[:])
			continue
		}
		if !helpers.IsActiveValidatorUsingTrie(val, currentEpoch) {
			// Inactive validator; treat it as missing.
			missingValidators = append(missingValidators, pubKey[:])
			continue
		}

		summary := validatorSummary[idx]
		pubKeys = append(pubKeys, pubKey[:])
		effectiveBalances = append(effectiveBalances, summary.CurrentEpochEffectiveBalance)
		beforeTransitionBalances = append(beforeTransitionBalances, summary.BeforeEpochTransitionBalance)
		afterTransitionBalances = append(afterTransitionBalances, summary.AfterEpochTransitionBalance)
		correctlyVotedTarget = append(correctlyVotedTarget, summary.IsPrevEpochTargetAttester)
		correctlyVotedHead = append(correctlyVotedHead, summary.IsPrevEpochHeadAttester)

		if headState.Version() == version.Phase0 {
			correctlyVotedSource = append(correctlyVotedSource, summary.IsPrevEpochAttester)
			inclusionSlots = append(inclusionSlots, summary.InclusionSlot)
			inclusionDistances = append(inclusionDistances, summary.InclusionDistance)
		} else {
			correctlyVotedSource = append(correctlyVotedSource, summary.IsPrevEpochSourceAttester)
			inactivityScores = append(inactivityScores, summary.InactivityScore)
		}
	}

	return &ethpb.ValidatorPerformanceResponse{
		PublicKeys:                    pubKeys,
		CorrectlyVotedSource:          correctlyVotedSource,
		CorrectlyVotedTarget:          correctlyVotedTarget, // In altair, when this is true then the attestation was definitely included.
		CorrectlyVotedHead:            correctlyVotedHead,
		CurrentEffectiveBalances:      effectiveBalances,
		BalancesBeforeEpochTransition: beforeTransitionBalances,
		BalancesAfterEpochTransition:  afterTransitionBalances,
		MissingValidators:             missingValidators,
		InclusionSlots:                inclusionSlots,     // Only populated in phase0
		InclusionDistances:            inclusionDistances, // Only populated in phase 0
		InactivityScores:              inactivityScores,   // Only populated in Altair
	}, nil
}

// GetIndividualVotes retrieves individual voting status of validators.
func (bs *Server) GetIndividualVotes(
	ctx context.Context,
	req *ethpb.IndividualVotesRequest,
) (*ethpb.IndividualVotesRespond, error) {
	currentEpoch := slots.ToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
	if req.Epoch > currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			errEpoch,
			currentEpoch,
			req.Epoch,
		)
	}

	s, err := slots.EpochEnd(req.Epoch)
	if err != nil {
		return nil, err
	}
	st, err := bs.ReplayerBuilder.ReplayerForSlot(s).ReplayBlocks(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to replay blocks for state at epoch %d: %v", req.Epoch, err)
	}
	// Track filtered validators to prevent duplication in the response.
	filtered := map[types.ValidatorIndex]bool{}
	filteredIndices := make([]types.ValidatorIndex, 0)
	votes := make([]*ethpb.IndividualVotesRespond_IndividualVote, 0, len(req.Indices)+len(req.PublicKeys))
	// Filter out assignments by public keys.
	for _, pubKey := range req.PublicKeys {
		index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
		if !ok {
			votes = append(votes, &ethpb.IndividualVotesRespond_IndividualVote{PublicKey: pubKey, ValidatorIndex: types.ValidatorIndex(^uint64(0))})
			continue
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
	sort.Slice(filteredIndices, func(i, j int) bool {
		return filteredIndices[i] < filteredIndices[j]
	})

	var v []*precompute.Validator
	var bal *precompute.Balance
	switch st.Version() {
	case version.Phase0:
		v, bal, err = precompute.New(ctx, st)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not set up pre compute instance: %v", err)
		}
		v, _, err = precompute.ProcessAttestations(ctx, st, v, bal)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not pre compute attestations: %v", err)
		}
	case version.Altair:
		v, bal, err = altair.InitializePrecomputeValidators(ctx, st)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not set up altair pre compute instance: %v", err)
		}
		v, _, err = altair.ProcessEpochParticipation(ctx, st, bal, v)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not pre compute attestations: %v", err)
		}
	default:
		return nil, status.Errorf(codes.Internal, "Invalid state type retrieved with a version of %d", st.Version())
	}

	for _, index := range filteredIndices {
		if uint64(index) >= uint64(len(v)) {
			votes = append(votes, &ethpb.IndividualVotesRespond_IndividualVote{ValidatorIndex: index})
			continue
		}
		val, err := st.ValidatorAtIndexReadOnly(index)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve validator: %v", err)

		}
		pb := val.PublicKey()
		votes = append(votes, &ethpb.IndividualVotesRespond_IndividualVote{
			Epoch:                            req.Epoch,
			PublicKey:                        pb[:],
			ValidatorIndex:                   index,
			IsSlashed:                        v[index].IsSlashed,
			IsWithdrawableInCurrentEpoch:     v[index].IsWithdrawableCurrentEpoch,
			IsActiveInCurrentEpoch:           v[index].IsActiveCurrentEpoch,
			IsActiveInPreviousEpoch:          v[index].IsActivePrevEpoch,
			IsCurrentEpochAttester:           v[index].IsCurrentEpochAttester,
			IsCurrentEpochTargetAttester:     v[index].IsCurrentEpochTargetAttester,
			IsPreviousEpochAttester:          v[index].IsPrevEpochAttester,
			IsPreviousEpochTargetAttester:    v[index].IsPrevEpochTargetAttester,
			IsPreviousEpochHeadAttester:      v[index].IsPrevEpochHeadAttester,
			CurrentEpochEffectiveBalanceGwei: v[index].CurrentEpochEffectiveBalance,
			InclusionSlot:                    v[index].InclusionSlot,
			InclusionDistance:                v[index].InclusionDistance,
			InactivityScore:                  v[index].InactivityScore,
		})
	}

	return &ethpb.IndividualVotesRespond{
		IndividualVotes: votes,
	}, nil
}

// Determines whether a validator has already exited.
func validatorHasExited(validator *ethpb.Validator, currentEpoch types.Epoch) bool {
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch
	if currentEpoch < validator.ActivationEligibilityEpoch {
		return false
	}
	if currentEpoch < validator.ActivationEpoch {
		return false
	}
	if validator.ExitEpoch == farFutureEpoch {
		return false
	}
	if currentEpoch < validator.ExitEpoch {
		if validator.Slashed {
			return false
		}
		return false
	}
	return true
}

func validatorStatus(validator *ethpb.Validator, epoch types.Epoch) ethpb.ValidatorStatus {
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch
	if validator == nil {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS
	}
	if epoch < validator.ActivationEligibilityEpoch {
		return ethpb.ValidatorStatus_DEPOSITED
	}
	if epoch < validator.ActivationEpoch {
		return ethpb.ValidatorStatus_PENDING
	}
	if validator.ExitEpoch == farFutureEpoch {
		return ethpb.ValidatorStatus_ACTIVE
	}
	if epoch < validator.ExitEpoch {
		if validator.Slashed {
			return ethpb.ValidatorStatus_SLASHING
		}
		return ethpb.ValidatorStatus_EXITING
	}
	return ethpb.ValidatorStatus_EXITED
}
