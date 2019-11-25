package beacon

import (
	"context"
	"sort"
	"strconv"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListValidatorBalances retrieves the validator balances for a given set of public keys.
// An optional Epoch parameter is provided to request historical validator balances from
// archived, persistent data.
func (bs *Server) ListValidatorBalances(
	ctx context.Context,
	req *ethpb.ListValidatorBalancesRequest) (*ethpb.ValidatorBalances, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	res := make([]*ethpb.ValidatorBalances_Balance, 0)
	filtered := map[uint64]bool{} // Track filtered validators to prevent duplication in the response.

	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}

	var requestingGenesis bool
	var epoch uint64
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListValidatorBalancesRequest_Epoch:
		epoch = q.Epoch
	case *ethpb.ListValidatorBalancesRequest_Genesis:
		requestingGenesis = q.Genesis
	default:
		epoch = helpers.CurrentEpoch(headState)
	}

	var balances []uint64
	validators := headState.Validators
	if requestingGenesis || epoch < helpers.CurrentEpoch(headState) {
		balances, err = bs.BeaconDB.ArchivedBalances(ctx, epoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve balances for epoch %d", epoch)
		}
		if balances == nil {
			return nil, status.Errorf(
				codes.NotFound,
				"Could not retrieve data for epoch %d, perhaps --archive in the running beacon node is disabled",
				0,
			)
		}
	} else if epoch == helpers.CurrentEpoch(headState) {
		balances = headState.Balances
	} else {
		// Otherwise, we are requesting data from the future and we return an error.
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			helpers.CurrentEpoch(headState),
			epoch,
		)
	}

	for _, pubKey := range req.PublicKeys {
		// Skip empty public key.
		if len(pubKey) == 0 {
			continue
		}

		index, ok, err := bs.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve validator index: %v", err)
		}
		if !ok {
			return nil, status.Errorf(codes.NotFound, "Could not find validator index for public key %#x", pubKey)
		}

		filtered[index] = true

		if int(index) >= len(balances) {
			return nil, status.Errorf(codes.OutOfRange, "Validator index %d >= balance list %d",
				index, len(balances))
		}

		res = append(res, &ethpb.ValidatorBalances_Balance{
			PublicKey: pubKey,
			Index:     index,
			Balance:   balances[index],
		})
	}

	for _, index := range req.Indices {
		if int(index) >= len(balances) {
			if epoch <= helpers.CurrentEpoch(headState) {
				return nil, status.Errorf(codes.OutOfRange, "Validator index %d does not exist in historical balances",
					index)
			}
			return nil, status.Errorf(codes.OutOfRange, "Validator index %d >= balance list %d",
				index, len(balances))
		}

		if !filtered[index] {
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: validators[index].PublicKey,
				Index:     index,
				Balance:   balances[index],
			})
		}
	}

	if len(req.Indices) == 0 && len(req.PublicKeys) == 0 {
		// Return everything.
		for i := 0; i < len(balances); i++ {
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: headState.Validators[i].PublicKey,
				Index:     uint64(i),
				Balance:   balances[i],
			})
		}
	}

	balancesCount := len(res)
	// If there are no balances, we simply return a response specifying this.
	// Otherwise, attempting to paginate 0 balances below would result in an error.
	if balancesCount == 0 {
		return &ethpb.ValidatorBalances{
			Epoch:         epoch,
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
	return &ethpb.ValidatorBalances{
		Epoch:         epoch,
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
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	currentEpoch := helpers.CurrentEpoch(headState)
	requestedEpoch := currentEpoch

	switch q := req.QueryFilter.(type) {
	case *ethpb.ListValidatorsRequest_Genesis:
		if q.Genesis {
			requestedEpoch = 0
		}
	case *ethpb.ListValidatorsRequest_Epoch:
		requestedEpoch = q.Epoch
	}

	vals := headState.Validators
	if requestedEpoch < currentEpoch {
		stopIdx := len(vals)
		for idx, val := range vals {
			// The first time we see a validator with an activation epoch > the requested epoch,
			// we know this validator is from the future relative to what the request wants.
			if val.ActivationEpoch > requestedEpoch {
				stopIdx = idx
				break
			}
		}
		vals = vals[:stopIdx]
	} else if requestedEpoch > currentEpoch {
		// Otherwise, we are requesting data from the future and we return an error.
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			currentEpoch,
			requestedEpoch,
		)
	}

	// Filter active validators if the request specifies it.
	res := vals
	if req.Active {
		filteredValidators := make([]*ethpb.Validator, 0)
		for _, val := range vals {
			if helpers.IsActiveValidator(val, requestedEpoch) {
				filteredValidators = append(filteredValidators, val)
			}
		}
		res = filteredValidators
	}

	validatorCount := len(res)
	// If there are no items, we simply return a response specifying this.
	// Otherwise, attempting to paginate 0 validators below would result in an error.
	if validatorCount == 0 {
		return &ethpb.Validators{
			Validators:    make([]*ethpb.Validator, 0),
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
		Validators:    res[start:end],
		TotalSize:     int32(validatorCount),
		NextPageToken: nextPageToken,
	}, nil
}

// GetValidatorActiveSetChanges retrieves the active set changes for a given epoch.
//
// This data includes any activations, voluntary exits, and involuntary
// ejections.
func (bs *Server) GetValidatorActiveSetChanges(
	ctx context.Context, req *ethpb.GetValidatorActiveSetChangesRequest,
) (*ethpb.ActiveSetChanges, error) {
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	currentEpoch := helpers.CurrentEpoch(headState)
	requestedEpoch := currentEpoch
	requestingGenesis := false

	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorActiveSetChangesRequest_Genesis:
		requestingGenesis = q.Genesis
		requestedEpoch = 0
	case *ethpb.GetValidatorActiveSetChangesRequest_Epoch:
		requestedEpoch = q.Epoch
	}

	activatedIndices := make([]uint64, 0)
	slashedIndices := make([]uint64, 0)
	exitedIndices := make([]uint64, 0)
	if requestingGenesis || requestedEpoch < currentEpoch {
		archivedChanges, err := bs.BeaconDB.ArchivedActiveValidatorChanges(ctx, requestedEpoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch archived active validator changes: %v", err)
		}
		if archivedChanges == nil {
			return nil, status.Errorf(
				codes.NotFound,
				"Did not find any data for epoch %d - perhaps no active set changed occurred during the epoch",
				requestedEpoch,
			)
		}
		activatedIndices = archivedChanges.Activated
		slashedIndices = archivedChanges.Slashed
		exitedIndices = archivedChanges.Exited
	} else if requestedEpoch == currentEpoch {
		activeValidatorCount, err := helpers.ActiveValidatorCount(headState, helpers.CurrentEpoch(headState))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
		}
		activatedIndices = validators.ActivatedValidatorIndices(helpers.CurrentEpoch(headState), headState.Validators)
		slashedIndices = validators.SlashedValidatorIndices(helpers.PrevEpoch(headState), headState.Validators)
		exitedIndices, err = validators.ExitedValidatorIndices(headState.Validators, activeValidatorCount)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not determine exited validator indices: %v", err)
		}
	} else {
		// We are requesting data from the future and we return an error.
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			currentEpoch,
			requestedEpoch,
		)
	}

	// We retrieve the public keys for the indices.
	activatedKeys := make([][]byte, len(activatedIndices))
	slashedKeys := make([][]byte, len(slashedIndices))
	exitedKeys := make([][]byte, len(exitedIndices))
	for i, idx := range activatedIndices {
		activatedKeys[i] = headState.Validators[idx].PublicKey
	}
	for i, idx := range slashedIndices {
		slashedKeys[i] = headState.Validators[idx].PublicKey
	}
	for i, idx := range exitedIndices {
		exitedKeys[i] = headState.Validators[idx].PublicKey
	}
	return &ethpb.ActiveSetChanges{
		Epoch:               requestedEpoch,
		ActivatedPublicKeys: activatedKeys,
		ExitedPublicKeys:    exitedKeys,
		SlashedPublicKeys:   slashedKeys,
	}, nil
}

// GetValidatorParticipation retrieves the validator participation information for a given epoch,
// it returns the information about validator's participation rate in voting on the proof of stake
// rules based on their balance compared to the total active validator balance.
func (bs *Server) GetValidatorParticipation(
	ctx context.Context, req *ethpb.GetValidatorParticipationRequest,
) (*ethpb.ValidatorParticipationResponse, error) {
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	currentEpoch := helpers.SlotToEpoch(headState.Slot)
	prevEpoch := helpers.PrevEpoch(headState)

	var requestedEpoch uint64
	var requestingGenesis bool
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorParticipationRequest_Genesis:
		requestingGenesis = q.Genesis
		requestedEpoch = 0
	case *ethpb.GetValidatorParticipationRequest_Epoch:
		requestedEpoch = q.Epoch
	default:
		requestedEpoch = prevEpoch
	}

	// If the request is from genesis or another past epoch, we look into our archived
	// data to find it and return it if it exists.
	if requestingGenesis || requestedEpoch < prevEpoch {
		participation, err := bs.BeaconDB.ArchivedValidatorParticipation(ctx, requestedEpoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch archived participation: %v", err)
		}
		if participation == nil {
			return nil, status.Errorf(
				codes.NotFound,
				"Could not retrieve data for epoch %d, perhaps --archive in the running beacon node is disabled",
				0,
			)
		}
		return &ethpb.ValidatorParticipationResponse{
			Epoch:         requestedEpoch,
			Finalized:     requestedEpoch <= headState.FinalizedCheckpoint.Epoch,
			Participation: participation,
		}, nil
	} else if requestedEpoch == currentEpoch {
		// We cannot retrieve participation for an epoch currently in progress.
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch currently in progress, current epoch %d, requesting %d",
			currentEpoch,
			requestedEpoch,
		)
	} else if requestedEpoch > currentEpoch {
		// We are requesting data from the future and we return an error.
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			currentEpoch,
			requestedEpoch,
		)
	}

	// Else if the request is for the current epoch, we compute validator participation
	// right away and return the result based on the head state.
	participation, err := epoch.ComputeValidatorParticipation(headState, requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute participation: %v", err)
	}
	return &ethpb.ValidatorParticipationResponse{
		Epoch:         currentEpoch,
		Finalized:     false, // The current epoch can never be finalized.
		Participation: participation,
	}, nil
}

// GetValidatorQueue retrieves the current validator queue information.
func (bs *Server) GetValidatorQueue(
	ctx context.Context, _ *ptypes.Empty,
) (*ethpb.ValidatorQueue, error) {
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	// Queue the validators whose eligible to activate and sort them by activation eligibility epoch number.
	// Additionally, determine those validators queued to exit
	awaitingExit := make([]uint64, 0)
	exitEpochs := make([]uint64, 0)
	activationQ := make([]uint64, 0)
	for idx, validator := range headState.Validators {
		eligibleActivated := validator.ActivationEligibilityEpoch != params.BeaconConfig().FarFutureEpoch
		canBeActive := validator.ActivationEpoch >= helpers.DelayedActivationExitEpoch(headState.FinalizedCheckpoint.Epoch)
		if eligibleActivated && canBeActive {
			activationQ = append(activationQ, uint64(idx))
		}
		if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochs = append(exitEpochs, validator.ExitEpoch)
			awaitingExit = append(awaitingExit, uint64(idx))
		}
	}
	sort.Slice(activationQ, func(i, j int) bool {
		return headState.Validators[i].ActivationEligibilityEpoch < headState.Validators[j].ActivationEligibilityEpoch
	})
	sort.Slice(awaitingExit, func(i, j int) bool {
		return headState.Validators[i].WithdrawableEpoch < headState.Validators[j].WithdrawableEpoch
	})

	// Only activate just enough validators according to the activation churn limit.
	activationQueueChurn := len(activationQ)
	activeValidatorCount, err := helpers.ActiveValidatorCount(headState, helpers.CurrentEpoch(headState))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
	}
	churnLimit, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute churn limit: %v", err)
	}

	exitQueueEpoch := uint64(0)
	for _, i := range exitEpochs {
		if exitQueueEpoch < i {
			exitQueueEpoch = i
		}
	}
	exitQueueChurn := 0
	for _, val := range headState.Validators {
		if val.ExitEpoch == exitQueueEpoch {
			exitQueueChurn++
		}
	}
	// Prevent churn limit from causing index out of bound issues.
	if int(churnLimit) < activationQueueChurn {
		activationQueueChurn = int(churnLimit)
	}
	if int(churnLimit) < exitQueueChurn {
		// If we are above the churn limit, we simply increase the churn by one.
		exitQueueEpoch++
		exitQueueChurn = int(churnLimit)
	}

	// We use the exit queue churn to determine if we have passed a churn limit.
	minEpoch := exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	exitQueueIndices := make([]uint64, 0)
	for _, valIdx := range awaitingExit {
		if headState.Validators[valIdx].WithdrawableEpoch < minEpoch {
			exitQueueIndices = append(exitQueueIndices, valIdx)
		}
	}

	// Get the public keys for the validators in the queues up to the allowed churn limits.
	activationQueueKeys := make([][]byte, len(activationQ))
	exitQueueKeys := make([][]byte, len(exitQueueIndices))
	for i, idx := range activationQ {
		activationQueueKeys[i] = headState.Validators[idx].PublicKey
	}
	for i, idx := range exitQueueIndices {
		exitQueueKeys[i] = headState.Validators[idx].PublicKey
	}

	return &ethpb.ValidatorQueue{
		ChurnLimit:           churnLimit,
		ActivationPublicKeys: activationQueueKeys,
		ExitPublicKeys:       exitQueueKeys,
	}, nil
}
