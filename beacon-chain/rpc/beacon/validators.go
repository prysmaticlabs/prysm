package beacon

import (
	"context"
	"sort"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
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
	req *ethpb.GetValidatorBalancesRequest) (*ethpb.ValidatorBalances, error) {

	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	res := make([]*ethpb.ValidatorBalances_Balance, 0)
	filtered := map[uint64]bool{} // track filtered validators to prevent duplication in the response.

	headState := bs.HeadFetcher.HeadState()
	var requestingGenesis bool
	var epoch uint64
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorBalancesRequest_Epoch:
		epoch = q.Epoch
	case *ethpb.GetValidatorBalancesRequest_Genesis:
		requestingGenesis = q.Genesis
	default:
		epoch = helpers.CurrentEpoch(headState)
	}

	var balances []uint64
	var err error
	validators := headState.Validators
	if requestingGenesis {
		balances, err = bs.BeaconDB.ArchivedBalances(ctx, 0 /* genesis epoch */)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "could not retrieve balances for epoch %d", epoch)
		}
	} else if !requestingGenesis && epoch < helpers.CurrentEpoch(headState) {
		balances, err = bs.BeaconDB.ArchivedBalances(ctx, epoch)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "could not retrieve balances for epoch %d", epoch)
		}
	} else {
		balances = headState.Balances
	}

	for _, pubKey := range req.PublicKeys {
		// Skip empty public key
		if len(pubKey) == 0 {
			continue
		}

		index, ok, err := bs.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
		}
		if !ok {
			return nil, status.Errorf(codes.Internal, "could not find validator index for public key  %#x not found", pubKey)
		}

		filtered[index] = true

		if int(index) >= len(balances) {
			return nil, status.Errorf(codes.OutOfRange, "validator index %d >= balance list %d",
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
				return nil, status.Errorf(codes.OutOfRange, "validator index %d does not exist in historical balances",
					index)
			}
			return nil, status.Errorf(codes.OutOfRange, "validator index %d >= balance list %d",
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
		// return everything.
		for i := 0; i < len(headState.Balances); i++ {
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: headState.Validators[i].PublicKey,
				Index:     uint64(i),
				Balance:   balances[i],
			})
		}
	}

	balancesCount := len(res)
	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), balancesCount)
	if err != nil {
		return nil, err
	}
	return &ethpb.ValidatorBalances{
		Epoch:         epoch,
		Balances:      res[start:end],
		TotalSize:     int32(balancesCount),
		NextPageToken: nextPageToken,
	}, nil
}

// GetValidators retrieves the current list of active validators with an optional historical epoch flag to
// to retrieve validator set in time.
func (bs *Server) GetValidators(
	ctx context.Context,
	req *ethpb.GetValidatorsRequest,
) (*ethpb.Validators, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	headState := bs.HeadFetcher.HeadState()
	requestedEpoch := helpers.CurrentEpoch(headState)
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorsRequest_Genesis:
		if q.Genesis {
			requestedEpoch = 0
		}
	case *ethpb.GetValidatorsRequest_Epoch:
		requestedEpoch = q.Epoch
	}

	finalizedEpoch := bs.FinalizationFetcher.FinalizedCheckpt().Epoch
	validators := headState.Validators
	if requestedEpoch < finalizedEpoch {
		stopIdx := len(validators)
		for idx, val := range validators {
			// The first time we see a validator with an activation epoch > the requested epoch,
			// we know this validator is from the future relative to what the request wants.
			if val.ActivationEpoch > requestedEpoch {
				stopIdx = idx
				break
			}
		}
		validators = validators[:stopIdx]
	}

	validatorCount := len(validators)
	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), validatorCount)
	if err != nil {
		return nil, err
	}

	return &ethpb.Validators{
		Validators:    validators[start:end],
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
	headState := bs.HeadFetcher.HeadState()
	requestedEpoch := helpers.CurrentEpoch(headState)
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorActiveSetChangesRequest_Genesis:
		if q.Genesis {
			requestedEpoch = 0
		}
	case *ethpb.GetValidatorActiveSetChangesRequest_Epoch:
		requestedEpoch = q.Epoch
	}

	activatedIndices := make([]uint64, 0)
	slashedIndices := make([]uint64, 0)
	exitedIndices := make([]uint64, 0)
	finalizedEpoch := bs.FinalizationFetcher.FinalizedCheckpt().Epoch
	var err error

	if requestedEpoch < finalizedEpoch {
		archivedChanges, err := bs.BeaconDB.ArchivedActiveValidatorChanges(ctx, requestedEpoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch archived active validator changes: %v", err)
		}
		activatedIndices = archivedChanges.Activated
		slashedIndices = archivedChanges.Slashed
		exitedIndices = archivedChanges.Exited
	} else {
		activatedIndices = validators.ActivatedValidatorIndices(headState)
		slashedIndices = validators.SlashedValidatorIndices(headState)
		exitedIndices, err = validators.ExitedValidatorIndices(headState)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not determine exited validator indices: %v", err)
		}
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
	headState := bs.HeadFetcher.HeadState()
	currentEpoch := helpers.SlotToEpoch(headState.Slot)
	prevEpoch := helpers.PrevEpoch(headState)

	var requestedEpoch uint64
	var isGenesis bool
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorParticipationRequest_Genesis:
		isGenesis = q.Genesis
	case *ethpb.GetValidatorParticipationRequest_Epoch:
		requestedEpoch = q.Epoch
	default:
		requestedEpoch = prevEpoch
	}

	if requestedEpoch > helpers.SlotToEpoch(headState.Slot) {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"cannot request data from an epoch in the future: req.Epoch %d, currentEpoch %d", requestedEpoch, currentEpoch,
		)
	}
	// If the request is from genesis or another past epoch, we look into our archived
	// data to find it and return it if it exists.
	if isGenesis {
		participation, err := bs.BeaconDB.ArchivedValidatorParticipation(ctx, 0)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch archived participation: %v", err)
		}
		if participation == nil {
			return nil, status.Error(codes.NotFound, "could not find archival data for epoch 0")
		}
		return &ethpb.ValidatorParticipationResponse{
			Epoch:         0,
			Finalized:     true,
			Participation: participation,
		}, nil
	} else if requestedEpoch < prevEpoch {
		participation, err := bs.BeaconDB.ArchivedValidatorParticipation(ctx, requestedEpoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch archived participation: %v", err)
		}
		if participation == nil {
			return nil, status.Errorf(codes.NotFound, "could not find archival data for epoch %d", requestedEpoch)
		}
		finalizedEpoch := bs.FinalizationFetcher.FinalizedCheckpt().Epoch
		// If the epoch we requested is <= the finalized epoch, we consider it finalized as well.
		finalized := requestedEpoch <= finalizedEpoch
		return &ethpb.ValidatorParticipationResponse{
			Epoch:         requestedEpoch,
			Finalized:     finalized,
			Participation: participation,
		}, nil
	}
	// Else if the request is for the current epoch, we compute validator participation
	// right away and return the result based on the head state.
	participation, err := epoch.ComputeValidatorParticipation(headState, requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not compute participation: %v", err)
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
	headState := bs.HeadFetcher.HeadState()
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
	churnLimit, err := helpers.ValidatorChurnLimit(headState)
	if err != nil {
		return nil, errors.Wrap(err, "could not get churn limit")
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
