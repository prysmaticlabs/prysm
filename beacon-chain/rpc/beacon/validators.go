package beacon

import (
	"context"
	"sort"
	"strconv"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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
	if int(req.PageSize) > flags.Get().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, flags.Get().MaxPageSize)
	}

	if !featureconfig.Get().NewStateMgmt {
		return bs.listValidatorsBalancesUsingOldArchival(ctx, req)
	}

	if bs.GenesisTimeFetcher == nil {
		return nil, status.Errorf(codes.Internal, "Nil genesis time fetcher")
	}
	currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
	requestedEpoch := currentEpoch
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListValidatorBalancesRequest_Epoch:
		requestedEpoch = q.Epoch
	case *ethpb.ListValidatorBalancesRequest_Genesis:
		requestedEpoch = 0
	default:
		requestedEpoch = currentEpoch
	}

	if requestedEpoch > currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			currentEpoch,
			requestedEpoch,
		)
	}
	res := make([]*ethpb.ValidatorBalances_Balance, 0)
	filtered := map[uint64]bool{} // Track filtered validators to prevent duplication in the response.

	requestedState, err := bs.StateGen.StateBySlot(ctx, helpers.StartSlot(requestedEpoch))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state")
	}

	validators := requestedState.Validators()
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
			return nil, status.Errorf(codes.NotFound, "Could not find validator index for public key %#x", pubkeyBytes)
		}

		filtered[index] = true

		if index >= uint64(len(balances)) {
			return nil, status.Errorf(codes.OutOfRange, "Validator index %d >= balance list %d",
				index, len(balances))
		}

		res = append(res, &ethpb.ValidatorBalances_Balance{
			PublicKey: pubKey,
			Index:     index,
			Balance:   balances[index],
		})
		balancesCount = len(res)
	}

	for _, index := range req.Indices {
		if index >= uint64(len(balances)) {
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
			pubkey := requestedState.PubkeyAtIndex(uint64(i))
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: pubkey[:],
				Index:     uint64(i),
				Balance:   balances[i],
			})
		}
		return &ethpb.ValidatorBalances{
			Epoch:         requestedEpoch,
			Balances:      res,
			TotalSize:     int32(balancesCount),
			NextPageToken: nextPageToken,
		}, nil
	}

	return &ethpb.ValidatorBalances{
		Epoch:         requestedEpoch,
		Balances:      res[start:end],
		TotalSize:     int32(balancesCount),
		NextPageToken: nextPageToken,
	}, nil
}

func (bs *Server) listValidatorsBalancesUsingOldArchival(
	ctx context.Context,
	req *ethpb.ListValidatorBalancesRequest) (*ethpb.ValidatorBalances, error) {
	res := make([]*ethpb.ValidatorBalances_Balance, 0)
	filtered := map[uint64]bool{} // Track filtered validators to prevent duplication in the response.

	if bs.HeadFetcher == nil {
		return nil, status.Error(codes.Internal, "Nil head state")
	}
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
	validators := headState.Validators()
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
		balances = headState.Balances()
	} else {
		// Otherwise, we are requesting data from the future and we return an error.
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			helpers.CurrentEpoch(headState),
			epoch,
		)
	}

	balancesCount := len(balances)
	for _, pubKey := range req.PublicKeys {
		// Skip empty public key.
		if len(pubKey) == 0 {
			continue
		}
		pubkeyBytes := bytesutil.ToBytes48(pubKey)
		index, ok := headState.ValidatorIndexByPubkey(pubkeyBytes)
		if !ok {
			return nil, status.Errorf(codes.NotFound, "Could not find validator index for public key %#x", pubkeyBytes)
		}

		filtered[index] = true

		if index >= uint64(len(balances)) {
			return nil, status.Errorf(codes.OutOfRange, "Validator index %d >= balance list %d",
				index, len(balances))
		}

		res = append(res, &ethpb.ValidatorBalances_Balance{
			PublicKey: pubKey,
			Index:     index,
			Balance:   balances[index],
		})
		balancesCount = len(res)
	}

	for _, index := range req.Indices {
		if index >= uint64(len(balances)) {
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

	if len(req.Indices) == 0 && len(req.PublicKeys) == 0 {
		// Return everything.
		for i := start; i < end; i++ {
			pubkey := headState.PubkeyAtIndex(uint64(i))
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: pubkey[:],
				Index:     uint64(i),
				Balance:   balances[i],
			})
		}
		return &ethpb.ValidatorBalances{
			Epoch:         epoch,
			Balances:      res,
			TotalSize:     int32(balancesCount),
			NextPageToken: nextPageToken,
		}, nil
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
	if int(req.PageSize) > flags.Get().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, flags.Get().MaxPageSize)
	}

	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
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
				"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
				currentEpoch,
				q.Epoch,
			)
		}
		requestedEpoch = q.Epoch
	}

	if helpers.StartSlot(requestedEpoch) > headState.Slot() {
		headState = headState.Copy()
		headState, err = state.ProcessSlots(ctx, headState, helpers.StartSlot(requestedEpoch))
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not process slots up to %d: %v",
				helpers.StartSlot(requestedEpoch),
				err,
			)
		}
	}

	validatorList := make([]*ethpb.Validators_ValidatorContainer, 0)

	for _, index := range req.Indices {
		val, err := headState.ValidatorAtIndex(index)
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not get validator")
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
		index, ok := headState.ValidatorIndexByPubkey(pubkeyBytes)
		if !ok {
			continue
		}
		val, err := headState.ValidatorAtIndex(index)
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not get validator")
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
		for i := 0; i < headState.NumValidators(); i++ {
			val, err := headState.ValidatorAtIndex(uint64(i))
			if err != nil {
				return nil, status.Error(codes.Internal, "Could not get validator")
			}
			validatorList = append(validatorList, &ethpb.Validators_ValidatorContainer{
				Index:     uint64(i),
				Validator: val,
			})
		}
	}

	if requestedEpoch < currentEpoch {
		stopIdx := len(validatorList)
		for idx, item := range validatorList {
			// The first time we see a validator with an activation epoch > the requested epoch,
			// we know this validator is from the future relative to what the request wants.
			if item.Validator.ActivationEpoch > requestedEpoch {
				stopIdx = idx
				break
			}
		}
		validatorList = validatorList[:stopIdx]
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
	var index uint64
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
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	if requestingIndex {
		if index >= uint64(headState.NumValidators()) {
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
	for i := 0; i < headState.NumValidators(); i++ {
		keyFromState := headState.PubkeyAtIndex(uint64(i))
		if keyFromState == pk48 {
			return headState.ValidatorAtIndex(uint64(i))
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

	if !featureconfig.Get().NewStateMgmt {
		return bs.getValidatorActiveSetChangesUsingOldArchival(ctx, req)
	}

	currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())

	var requestedEpoch uint64
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
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			currentEpoch,
			requestedEpoch,
		)
	}

	activatedIndices := make([]uint64, 0)
	exitedIndices := make([]uint64, 0)
	slashedIndices := make([]uint64, 0)
	ejectedIndices := make([]uint64, 0)

	requestedState, err := bs.StateGen.StateBySlot(ctx, helpers.StartSlot(requestedEpoch))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}

	activeValidatorCount, err := helpers.ActiveValidatorCount(requestedState, helpers.CurrentEpoch(requestedState))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
	}
	vs := requestedState.Validators()
	activatedIndices = validators.ActivatedValidatorIndices(helpers.CurrentEpoch(requestedState), vs)
	exitedIndices, err = validators.ExitedValidatorIndices(helpers.CurrentEpoch(requestedState), vs, activeValidatorCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine exited validator indices: %v", err)
	}
	slashedIndices = validators.SlashedValidatorIndices(helpers.CurrentEpoch(requestedState), vs)
	ejectedIndices, err = validators.EjectedValidatorIndices(helpers.CurrentEpoch(requestedState), vs, activeValidatorCount)
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

func (bs *Server) getValidatorActiveSetChangesUsingOldArchival(
	ctx context.Context, req *ethpb.GetValidatorActiveSetChangesRequest,
) (*ethpb.ActiveSetChanges, error) {
	if bs.HeadFetcher == nil {
		return nil, status.Error(codes.Internal, "Nil head state")
	}
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	if bs.BeaconDB == nil {
		return nil, status.Error(codes.Internal, "Nil beacon DB")
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
	exitedIndices := make([]uint64, 0)
	slashedIndices := make([]uint64, 0)
	ejectedIndices := make([]uint64, 0)
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
		exitedIndices = archivedChanges.Exited
		slashedIndices = archivedChanges.Slashed
		ejectedIndices = archivedChanges.Ejected
	} else if requestedEpoch == currentEpoch {
		activeValidatorCount, err := helpers.ActiveValidatorCount(headState, helpers.PrevEpoch(headState))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
		}
		vals := headState.Validators()
		activatedIndices = validators.ActivatedValidatorIndices(helpers.PrevEpoch(headState), vals)
		exitedIndices, err = validators.ExitedValidatorIndices(helpers.PrevEpoch(headState), vals, activeValidatorCount)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not determine exited validator indices: %v", err)
		}
		slashedIndices = validators.SlashedValidatorIndices(helpers.PrevEpoch(headState), vals)
		ejectedIndices, err = validators.EjectedValidatorIndices(helpers.PrevEpoch(headState), vals, activeValidatorCount)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not determine ejected validator indices: %v", err)
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
	exitedKeys := make([][]byte, len(exitedIndices))
	slashedKeys := make([][]byte, len(slashedIndices))
	ejectedKeys := make([][]byte, len(ejectedIndices))
	for i, idx := range activatedIndices {
		pubkey := headState.PubkeyAtIndex(idx)
		activatedKeys[i] = pubkey[:]
	}
	for i, idx := range exitedIndices {
		pubkey := headState.PubkeyAtIndex(idx)
		exitedKeys[i] = pubkey[:]
	}
	for i, idx := range slashedIndices {
		pubkey := headState.PubkeyAtIndex(idx)
		slashedKeys[i] = pubkey[:]
	}
	for i, idx := range ejectedIndices {
		pubkey := headState.PubkeyAtIndex(idx)
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

	if !featureconfig.Get().NewStateMgmt {
		return bs.getValidatorParticipationUsingOldArchival(ctx, req)
	}

	currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())

	var requestedEpoch uint64
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorParticipationRequest_Genesis:
		requestedEpoch = 0
	case *ethpb.GetValidatorParticipationRequest_Epoch:
		requestedEpoch = q.Epoch
	default:
		// Prevent underflow and ensure participation is always queried for previous epoch.
		if currentEpoch > 1 {
			requestedEpoch = currentEpoch - 1
		}
	}

	if requestedEpoch >= currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch until older than current epoch, current epoch %d, requesting %d",
			currentEpoch,
			requestedEpoch,
		)
	}

	requestedState, err := bs.StateGen.StateBySlot(ctx, helpers.StartSlot(requestedEpoch+1))
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get state")
	}

	v, b, err := precompute.New(ctx, requestedState)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not set up pre compute instance")
	}
	_, b, err = precompute.ProcessAttestations(ctx, requestedState, v, b)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not pre compute attestations")
	}

	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}

	return &ethpb.ValidatorParticipationResponse{
		Epoch:     requestedEpoch,
		Finalized: requestedEpoch <= headState.FinalizedCheckpointEpoch(),
		Participation: &ethpb.ValidatorParticipation{
			GlobalParticipationRate: float32(b.PrevEpochTargetAttested) / float32(b.ActivePrevEpoch),
			VotedEther:              b.PrevEpochTargetAttested,
			EligibleEther:           b.ActivePrevEpoch,
		},
	}, nil
}

func (bs *Server) getValidatorParticipationUsingOldArchival(
	ctx context.Context, req *ethpb.GetValidatorParticipationRequest,
) (*ethpb.ValidatorParticipationResponse, error) {
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}

	currentEpoch := helpers.CurrentEpoch(headState)
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
			Finalized:     requestedEpoch <= headState.FinalizedCheckpointEpoch(),
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

	pBal := bs.ParticipationFetcher.Participation(requestedEpoch)
	if pBal == nil {
		return nil, status.Errorf(codes.Unavailable, "Participation information for epoch %d is not yet available", requestedEpoch)
	}
	participation := &ethpb.ValidatorParticipation{
		EligibleEther: pBal.ActivePrevEpoch,
		VotedEther:    pBal.PrevEpochTargetAttested,
	}
	participation.GlobalParticipationRate = float32(0)
	// only divide if prevEpoch is non zero
	if pBal.ActivePrevEpoch != 0 {
		participation.GlobalParticipationRate = float32(float64(pBal.PrevEpochTargetAttested) / float64(pBal.ActivePrevEpoch))
	}

	return &ethpb.ValidatorParticipationResponse{
		Epoch:         requestedEpoch,
		Finalized:     requestedEpoch <= headState.FinalizedCheckpointEpoch(),
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
	vals := headState.Validators()
	for idx, validator := range vals {
		eligibleActivated := validator.ActivationEligibilityEpoch != params.BeaconConfig().FarFutureEpoch
		canBeActive := validator.ActivationEpoch >= helpers.ActivationExitEpoch(headState.FinalizedCheckpointEpoch())
		if eligibleActivated && canBeActive {
			activationQ = append(activationQ, uint64(idx))
		}
		if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochs = append(exitEpochs, validator.ExitEpoch)
			awaitingExit = append(awaitingExit, uint64(idx))
		}
	}
	sort.Slice(activationQ, func(i, j int) bool {
		return vals[i].ActivationEligibilityEpoch < vals[j].ActivationEligibilityEpoch
	})
	sort.Slice(awaitingExit, func(i, j int) bool {
		return vals[i].WithdrawableEpoch < vals[j].WithdrawableEpoch
	})

	// Only activate just enough validators according to the activation churn limit.
	activationQueueChurn := uint64(len(activationQ))
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
	exitQueueChurn := uint64(0)
	for _, val := range vals {
		if val.ExitEpoch == exitQueueEpoch {
			exitQueueChurn++
		}
	}
	// Prevent churn limit from causing index out of bound issues.
	if churnLimit < activationQueueChurn {
		activationQueueChurn = churnLimit
	}
	if churnLimit < exitQueueChurn {
		// If we are above the churn limit, we simply increase the churn by one.
		exitQueueEpoch++
		exitQueueChurn = churnLimit
	}

	// We use the exit queue churn to determine if we have passed a churn limit.
	minEpoch := exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	exitQueueIndices := make([]uint64, 0)
	for _, valIdx := range awaitingExit {
		val := vals[valIdx]
		// Ensure the validator has not yet exited before adding its index to the exit queue.
		if val.WithdrawableEpoch < minEpoch && !validatorHasExited(val, helpers.CurrentEpoch(headState)) {
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

	if bs.GenesisTimeFetcher.CurrentSlot() > headState.Slot() {
		headState, err = state.ProcessSlots(ctx, headState, bs.GenesisTimeFetcher.CurrentSlot())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots: %v", err)
		}
	}
	vp, bp, err := precompute.New(ctx, headState)
	if err != nil {
		return nil, err
	}
	vp, bp, err = precompute.ProcessAttestations(ctx, headState, vp, bp)
	if err != nil {
		return nil, err
	}
	headState, err = precompute.ProcessRewardsAndPenaltiesPrecompute(headState, bp, vp)
	if err != nil {
		return nil, err
	}
	validatorSummary := vp

	responseCap := len(req.Indices) + len(req.PublicKeys)
	validatorIndices := make([]uint64, 0, responseCap)
	missingValidators := make([][]byte, 0, responseCap)

	filtered := map[uint64]bool{} // Track filtered validators to prevent duplication in the response.
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

	currentEpoch := helpers.CurrentEpoch(headState)
	responseCap = len(validatorIndices)
	pubKeys := make([][]byte, 0, responseCap)
	beforeTransitionBalances := make([]uint64, 0, responseCap)
	afterTransitionBalances := make([]uint64, 0, responseCap)
	effectiveBalances := make([]uint64, 0, responseCap)
	inclusionSlots := make([]uint64, 0, responseCap)
	inclusionDistances := make([]uint64, 0, responseCap)
	correctlyVotedSource := make([]bool, 0, responseCap)
	correctlyVotedTarget := make([]bool, 0, responseCap)
	correctlyVotedHead := make([]bool, 0, responseCap)
	// Append performance summaries.
	// Also track missing validators using public keys.
	for _, idx := range validatorIndices {
		val, err := headState.ValidatorAtIndexReadOnly(idx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not get validator: %v", err)
		}
		pubKey := val.PublicKey()
		if idx >= uint64(len(validatorSummary)) {
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
		inclusionSlots = append(inclusionSlots, summary.InclusionSlot)
		inclusionDistances = append(inclusionDistances, summary.InclusionDistance)
		correctlyVotedSource = append(correctlyVotedSource, summary.IsPrevEpochAttester)
		correctlyVotedTarget = append(correctlyVotedTarget, summary.IsPrevEpochTargetAttester)
		correctlyVotedHead = append(correctlyVotedHead, summary.IsPrevEpochHeadAttester)
	}

	return &ethpb.ValidatorPerformanceResponse{
		PublicKeys:                    pubKeys,
		InclusionSlots:                inclusionSlots,
		InclusionDistances:            inclusionDistances,
		CorrectlyVotedSource:          correctlyVotedSource,
		CorrectlyVotedTarget:          correctlyVotedTarget,
		CorrectlyVotedHead:            correctlyVotedHead,
		CurrentEffectiveBalances:      effectiveBalances,
		BalancesBeforeEpochTransition: beforeTransitionBalances,
		BalancesAfterEpochTransition:  afterTransitionBalances,
		MissingValidators:             missingValidators,
	}, nil
}

// GetIndividualVotes retrieves individual voting status of validators.
func (bs *Server) GetIndividualVotes(
	ctx context.Context,
	req *ethpb.IndividualVotesRequest,
) (*ethpb.IndividualVotesRespond, error) {
	if !featureconfig.Get().NewStateMgmt {
		return nil, status.Error(codes.FailedPrecondition, "Requires --enable-new-state-mgmt to function")
	}

	currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
	if req.Epoch > currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			currentEpoch,
			req.Epoch,
		)
	}

	requestedState, err := bs.StateGen.StateBySlot(ctx, helpers.StartSlot(req.Epoch))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve archived state for epoch %d: %v", req.Epoch, err)
	}
	// Track filtered validators to prevent duplication in the response.
	filtered := map[uint64]bool{}
	filteredIndices := make([]uint64, 0)
	votes := make([]*ethpb.IndividualVotesRespond_IndividualVote, 0, len(req.Indices)+len(req.PublicKeys))
	// Filter out assignments by public keys.
	for _, pubKey := range req.PublicKeys {
		index, ok := requestedState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
		if !ok {
			votes = append(votes, &ethpb.IndividualVotesRespond_IndividualVote{PublicKey: pubKey, ValidatorIndex: ^uint64(0)})
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

	v, b, err := precompute.New(ctx, requestedState)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not set up pre compute instance: %v", err)
	}
	v, b, err = precompute.ProcessAttestations(ctx, requestedState, v, b)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not pre compute attestations")
	}
	vals := requestedState.ValidatorsReadOnly()
	for _, index := range filteredIndices {
		if index >= uint64(len(v)) {
			votes = append(votes, &ethpb.IndividualVotesRespond_IndividualVote{ValidatorIndex: index})
			continue
		}
		pb := vals[index].PublicKey()
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
		})
	}

	return &ethpb.IndividualVotesRespond{
		IndividualVotes: votes,
	}, nil
}

// Determines whether a validator has already exited.
func validatorHasExited(validator *ethpb.Validator, currentEpoch uint64) bool {
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
