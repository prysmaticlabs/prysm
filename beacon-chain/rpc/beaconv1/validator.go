package beaconv1

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/statefetcher"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	"github.com/prysmaticlabs/prysm/proto/migration"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// validatorNotFoundError represents an error scenario where a validator could not be found.
type validatorNotFoundError struct {
	message string
}

// newValidatorNotFoundError creates a new error instance.
func newValidatorNotFoundError(validatorId []byte) validatorNotFoundError {
	return validatorNotFoundError{
		message: fmt.Sprintf("could not find validator with public key '%#x'", validatorId),
	}
}

// Error returns the underlying error message.
func (e *validatorNotFoundError) Error() string {
	return e.message
}

// invalidValidatorIdError represents an error scenario where a validator's ID is invalid.
type invalidValidatorIdError struct {
	message string
}

// newValidatorNotFoundError creates a new error instance.
func newInvalidValidatorIdError(validatorId []byte, reason error) invalidValidatorIdError {
	return invalidValidatorIdError{
		message: fmt.Errorf("could not decode validator id '%s': %w", string(validatorId), reason).Error(),
	}
}

// Error returns the underlying error message.
func (e *invalidValidatorIdError) Error() string {
	return e.message
}

// GetValidator returns a validator specified by state and id or public key along with status and balance.
func (bs *Server) GetValidator(ctx context.Context, req *ethpb.StateValidatorRequest) (*ethpb.StateValidatorResponse, error) {
	state, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		if stateNotFoundErr, ok := err.(*statefetcher.StateNotFoundError); ok {
			return nil, status.Errorf(codes.NotFound, "could not get state: %v", stateNotFoundErr)
		} else if errors.Is(err, statefetcher.ErrInvalidStateId) {
			return nil, status.Errorf(codes.InvalidArgument, "could not get state: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "State not found: %v", err)
	}
	if len(req.ValidatorId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Validator ID is required")
	}
	valContainer, err := valContainersByRequestIds(state, [][]byte{req.ValidatorId})
	if err != nil {
		return nil, handleValContainerErr(err)
	}
	return &ethpb.StateValidatorResponse{Data: valContainer[0]}, nil
}

// ListValidators returns filterable list of validators with their balance, status and index.
// TODO(#8901): missing status support.
func (bs *Server) ListValidators(ctx context.Context, req *ethpb.StateValidatorsRequest) (*ethpb.StateValidatorsResponse, error) {
	state, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		if stateNotFoundErr, ok := err.(*statefetcher.StateNotFoundError); ok {
			return nil, status.Errorf(codes.NotFound, "State not found: %v", stateNotFoundErr)
		} else if errors.Is(err, statefetcher.ErrInvalidStateId) {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid state ID: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}

	valContainers, err := valContainersByRequestIds(state, req.Id)
	if err != nil {
		return nil, handleValContainerErr(err)
	}
	return &ethpb.StateValidatorsResponse{Data: valContainers}, nil
}

// ListValidatorBalances returns a filterable list of validator balances.
func (bs *Server) ListValidatorBalances(ctx context.Context, req *ethpb.ValidatorBalancesRequest) (*ethpb.ValidatorBalancesResponse, error) {
	state, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		if stateNotFoundErr, ok := err.(*statefetcher.StateNotFoundError); ok {
			return nil, status.Errorf(codes.NotFound, "State not found: %v", stateNotFoundErr)
		} else if errors.Is(err, statefetcher.ErrInvalidStateId) {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid state ID: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}

	valContainers, err := valContainersByRequestIds(state, req.Id)
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
	return &ethpb.ValidatorBalancesResponse{Data: valBalances}, nil
}

// ListCommittees retrieves the committees for the given state at the given epoch.
// If the requested slot and index are defined, only those committees are returned.
func (bs *Server) ListCommittees(ctx context.Context, req *ethpb.StateCommitteesRequest) (*ethpb.StateCommitteesResponse, error) {
	state, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		if stateNotFoundErr, ok := err.(*statefetcher.StateNotFoundError); ok {
			return nil, status.Errorf(codes.NotFound, "State not found: %v", stateNotFoundErr)
		} else if errors.Is(err, statefetcher.ErrInvalidStateId) {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid state ID: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}

	epoch := helpers.SlotToEpoch(state.Slot())
	if req.Epoch != nil {
		epoch = *req.Epoch
	}
	activeCount, err := helpers.ActiveValidatorCount(state, epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
	}

	startSlot, err := helpers.StartSlot(epoch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid epoch: %v", err)
	}
	endSlot, err := helpers.EndSlot(epoch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid epoch: %v", err)
	}
	committeesPerSlot := helpers.SlotCommitteeCount(activeCount)
	committees := make([]*ethpb.Committee, 0)
	for slot := startSlot; slot <= endSlot; slot++ {
		if req.Slot != nil && slot != *req.Slot {
			continue
		}
		for index := types.CommitteeIndex(0); index < types.CommitteeIndex(committeesPerSlot); index++ {
			if req.Index != nil && index != *req.Index {
				continue
			}
			committee, err := helpers.BeaconCommitteeFromState(state, slot, index)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get committee: %v", err)
			}
			committeeContainer := &ethpb.Committee{
				Index:      index,
				Slot:       slot,
				Validators: committee,
			}
			committees = append(committees, committeeContainer)
		}
	}
	return &ethpb.StateCommitteesResponse{Data: committees}, nil
}

// This function returns the validator object based on the passed in ID. The validator ID could be its public key,
// or its index.
func valContainersByRequestIds(state iface.BeaconState, validatorIds [][]byte) ([]*ethpb.ValidatorContainer, error) {
	var valContainers []*ethpb.ValidatorContainer
	if len(validatorIds) == 0 {
		allValidators := state.Validators()
		allBalances := state.Balances()
		valContainers = make([]*ethpb.ValidatorContainer, len(allValidators))
		for i, validator := range allValidators {
			valContainers[i] = &ethpb.ValidatorContainer{
				Index:     types.ValidatorIndex(i),
				Balance:   allBalances[i],
				Validator: migration.V1Alpha1ValidatorToV1(validator),
			}
		}
	} else {
		valContainers = make([]*ethpb.ValidatorContainer, len(validatorIds))
		for i, validatorId := range validatorIds {
			var valIndex types.ValidatorIndex
			if len(validatorId) == params.BeaconConfig().BLSPubkeyLength {
				var ok bool
				valIndex, ok = state.ValidatorIndexByPubkey(bytesutil.ToBytes48(validatorId))
				if !ok {
					e := newValidatorNotFoundError(validatorId)
					return nil, &e
				}
			} else {
				index, err := strconv.ParseUint(string(validatorId), 10, 64)
				if err != nil {
					e := newInvalidValidatorIdError(validatorId, err)
					return nil, &e
				}
				valIndex = types.ValidatorIndex(index)
			}
			validator, err := state.ValidatorAtIndex(valIndex)
			if outOfRangeErr, ok := err.(*stateV0.IndexOutOfRangeError); ok {
				return nil, outOfRangeErr
			}
			if err != nil {
				return nil, fmt.Errorf("could not get validator: %w", err)
			}

			valContainers[i] = &ethpb.ValidatorContainer{
				Index:     valIndex,
				Balance:   validator.EffectiveBalance,
				Validator: migration.V1Alpha1ValidatorToV1(validator),
			}
		}
	}
	return valContainers, nil
}

func handleValContainerErr(err error) error {
	if outOfRangeErr, ok := err.(*stateV0.IndexOutOfRangeError); ok {
		return status.Errorf(codes.InvalidArgument, "Invalid validator ID: %v", outOfRangeErr)
	}
	if invalidIdErr, ok := err.(*invalidValidatorIdError); ok {
		return status.Errorf(codes.InvalidArgument, "Invalid validator ID: %v", invalidIdErr)
	}
	if notFoundErr, ok := err.(*validatorNotFoundError); ok {
		return status.Errorf(codes.NotFound, "Validator not found: %v", notFoundErr)
	}
	return status.Errorf(codes.Internal, "Could not get validator container: %v", err)
}
