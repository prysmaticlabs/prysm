package beacon

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	statenative "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// invalidValidatorIdError represents an error scenario where a validator's ID is invalid.
type invalidValidatorIdError struct {
	message string
}

// newInvalidValidatorIdError creates a new error instance.
func newInvalidValidatorIdError(validatorId []byte, reason error) invalidValidatorIdError {
	return invalidValidatorIdError{
		message: errors.Wrapf(reason, "could not decode validator id '%s'", string(validatorId)).Error(),
	}
}

// Error returns the underlying error message.
func (e *invalidValidatorIdError) Error() string {
	return e.message
}

// GetValidator returns a validator specified by state and id or public key along with status and balance.
func (bs *Server) GetValidator(ctx context.Context, req *ethpb.StateValidatorRequest) (*ethpb.StateValidatorResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetValidator")
	defer span.End()

	st, err := bs.Stater.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}
	if len(req.ValidatorId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Validator ID is required")
	}
	valContainer, err := valContainersByRequestIds(st, [][]byte{req.ValidatorId})
	if err != nil {
		return nil, handleValContainerErr(err)
	}
	if len(valContainer) == 0 {
		return nil, status.Error(codes.NotFound, "Could not find validator")
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, req.StateId, bs.OptimisticModeFetcher, bs.Stater, bs.ChainInfoFetcher, bs.BeaconDB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate root of latest block header")
	}
	isFinalized := bs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	return &ethpb.StateValidatorResponse{Data: valContainer[0], ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
}

// ListValidators returns filterable list of validators with their balance, status and index.
func (bs *Server) ListValidators(ctx context.Context, req *ethpb.StateValidatorsRequest) (*ethpb.StateValidatorsResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListValidators")
	defer span.End()

	st, err := bs.Stater.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	valContainers, err := valContainersByRequestIds(st, req.Id)
	if err != nil {
		return nil, handleValContainerErr(err)
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, req.StateId, bs.OptimisticModeFetcher, bs.Stater, bs.ChainInfoFetcher, bs.BeaconDB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate root of latest block header")
	}
	isFinalized := bs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	// Exit early if no matching validators we found or we don't want to further filter validators by status.
	if len(valContainers) == 0 || len(req.Status) == 0 {
		return &ethpb.StateValidatorsResponse{Data: valContainers, ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
	}

	filterStatus := make(map[validator.ValidatorStatus]bool, len(req.Status))
	const lastValidStatusValue = 12
	for _, ss := range req.Status {
		if ss > lastValidStatusValue {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid status "+ss.String())
		}
		filterStatus[validator.ValidatorStatus(ss)] = true
	}
	epoch := slots.ToEpoch(st.Slot())
	filteredVals := make([]*ethpb.ValidatorContainer, 0, len(valContainers))
	for _, vc := range valContainers {
		readOnlyVal, err := statenative.NewValidator(migration.V1ValidatorToV1Alpha1(vc.Validator))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not convert validator: %v", err)
		}
		valStatus, err := helpers.ValidatorStatus(readOnlyVal, epoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator status: %v", err)
		}
		valSubStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator sub status: %v", err)
		}
		if filterStatus[valStatus] || filterStatus[valSubStatus] {
			filteredVals = append(filteredVals, vc)
		}
	}

	return &ethpb.StateValidatorsResponse{Data: filteredVals, ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
}

// ListValidatorBalances returns a filterable list of validator balances.
func (bs *Server) ListValidatorBalances(ctx context.Context, req *ethpb.ValidatorBalancesRequest) (*ethpb.ValidatorBalancesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListValidatorBalances")
	defer span.End()

	st, err := bs.Stater.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	valContainers, err := valContainersByRequestIds(st, req.Id)
	if err != nil {
		return nil, handleValContainerErr(err)
	}
	valBalances := make([]*ethpb.ValidatorBalance, len(valContainers))
	for i := 0; i < len(valContainers); i++ {
		valBalances[i] = &ethpb.ValidatorBalance{
			Index:   valContainers[i].Index,
			Balance: valContainers[i].Balance,
		}
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, req.StateId, bs.OptimisticModeFetcher, bs.Stater, bs.ChainInfoFetcher, bs.BeaconDB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate root of latest block header")
	}
	isFinalized := bs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	return &ethpb.ValidatorBalancesResponse{Data: valBalances, ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
}

// This function returns the validator object based on the passed in ID. The validator ID could be its public key,
// or its index.
func valContainersByRequestIds(state state.BeaconState, validatorIds [][]byte) ([]*ethpb.ValidatorContainer, error) {
	epoch := slots.ToEpoch(state.Slot())
	var valContainers []*ethpb.ValidatorContainer
	allBalances := state.Balances()
	if len(validatorIds) == 0 {
		allValidators := state.Validators()
		valContainers = make([]*ethpb.ValidatorContainer, len(allValidators))
		for i, val := range allValidators {
			readOnlyVal, err := statenative.NewValidator(val)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not convert validator: %v", err)
			}
			subStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
			if err != nil {
				return nil, errors.Wrap(err, "could not get validator sub status")
			}
			valContainers[i] = &ethpb.ValidatorContainer{
				Index:     primitives.ValidatorIndex(i),
				Balance:   allBalances[i],
				Status:    ethpb.ValidatorStatus(subStatus),
				Validator: migration.V1Alpha1ValidatorToV1(val),
			}
		}
	} else {
		valContainers = make([]*ethpb.ValidatorContainer, 0, len(validatorIds))
		for _, validatorId := range validatorIds {
			var valIndex primitives.ValidatorIndex
			if len(validatorId) == params.BeaconConfig().BLSPubkeyLength {
				var ok bool
				valIndex, ok = state.ValidatorIndexByPubkey(bytesutil.ToBytes48(validatorId))
				if !ok {
					// Ignore well-formed yet unknown public keys.
					continue
				}
			} else {
				index, err := strconv.ParseUint(string(validatorId), 10, 64)
				if err != nil {
					e := newInvalidValidatorIdError(validatorId, err)
					return nil, &e
				}
				valIndex = primitives.ValidatorIndex(index)
			}
			val, err := state.ValidatorAtIndex(valIndex)
			if _, ok := err.(*statenative.ValidatorIndexOutOfRangeError); ok {
				// Ignore well-formed yet unknown indexes.
				continue
			}
			if err != nil {
				return nil, errors.Wrap(err, "could not get validator")
			}
			v1Validator := migration.V1Alpha1ValidatorToV1(val)
			readOnlyVal, err := statenative.NewValidator(val)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not convert validator: %v", err)
			}
			subStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
			if err != nil {
				return nil, errors.Wrap(err, "could not get validator sub status")
			}
			valContainers = append(valContainers, &ethpb.ValidatorContainer{
				Index:     valIndex,
				Balance:   allBalances[valIndex],
				Status:    ethpb.ValidatorStatus(subStatus),
				Validator: v1Validator,
			})
		}
	}

	return valContainers, nil
}

func handleValContainerErr(err error) error {
	if outOfRangeErr, ok := err.(*statenative.ValidatorIndexOutOfRangeError); ok {
		return status.Errorf(codes.InvalidArgument, "Invalid validator ID: %v", outOfRangeErr)
	}
	if invalidIdErr, ok := err.(*invalidValidatorIdError); ok {
		return status.Errorf(codes.InvalidArgument, "Invalid validator ID: %v", invalidIdErr)
	}
	return status.Errorf(codes.Internal, "Could not get validator container: %v", err)
}
