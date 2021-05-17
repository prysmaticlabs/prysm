package beaconv1

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/proto/migration"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetValidator returns a validator specified by state and id or public key along with status and balance.
func (bs *Server) GetValidator(ctx context.Context, req *ethpb.StateValidatorRequest) (*ethpb.StateValidatorResponse, error) {
	state, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}
	valContainer, err := valContainerById(state, req.ValidatorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}
	return &ethpb.StateValidatorResponse{Data: valContainer}, nil
}

// ListValidators returns filterable list of validators with their balance, status and index.
// NOTE: missing status support.
func (bs *Server) ListValidators(ctx context.Context, req *ethpb.StateValidatorsRequest) (*ethpb.StateValidatorsResponse, error) {
	state, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}

	valContainers := make([]*ethpb.ValidatorContainer, len(req.Id))
	for i := 0; i < len(req.Id); i++ {
		valContainer, err := valContainerById(state, req.Id[i])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
		}
		valContainers[i] = valContainer
	}
	return &ethpb.StateValidatorsResponse{Data: valContainers}, nil
}

// ListValidatorBalances returns a filterable list of validator balances.
func (bs *Server) ListValidatorBalances(ctx context.Context, req *ethpb.ValidatorBalancesRequest) (*ethpb.ValidatorBalancesResponse, error) {
	state, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}

	valBalances := make([]*ethpb.ValidatorBalance, len(req.Id))
	for i := 0; i < len(req.Id); i++ {
		_, valIndex, err := validatorById(state, req.Id[i])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator: %v", err)
		}
		bal, err := state.BalanceAtIndex(valIndex)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
		}
		valBalances[i] = &ethpb.ValidatorBalance{
			Index:   valIndex,
			Balance: bal,
		}
	}
	return &ethpb.ValidatorBalancesResponse{Data: valBalances}, nil
}

// ListCommittees retrieves the committees for the given state at the given epoch.
// If the requested slot and index are defined, only those committees are returned.
func (bs *Server) ListCommittees(ctx context.Context, req *ethpb.StateCommitteesRequest) (*ethpb.StateCommitteesResponse, error) {
	state, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
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
		return nil, status.Errorf(codes.Internal, "Could not get epoch start slot: %v", err)
	}
	endSlot, err := helpers.EndSlot(epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get epoch end slot: %v", err)
	}
	committeesPerSlot := helpers.SlotCommitteeCount(activeCount)
	committees := make([]*ethpb.Committee, 0)
	for slot := startSlot; slot <= endSlot; slot++ {
		for index := types.CommitteeIndex(0); index < types.CommitteeIndex(committeesPerSlot); index++ {
			if req.Slot != nil && slot != *req.Slot {
				continue
			}
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

func valContainerById(state iface.BeaconState, validatorId []byte) (*ethpb.ValidatorContainer, error) {
	val, valIndex, err := validatorById(state, validatorId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get validator: %v", err)
	}

	bal, err := state.BalanceAtIndex(valIndex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get validator balance: %v", err)
	}
	valContainer := &ethpb.ValidatorContainer{
		Index:     valIndex,
		Balance:   bal,
		Status:    "",
		Validator: val,
	}
	return valContainer, nil
}

func validatorById(state iface.BeaconState, validatorId []byte) (*ethpb.Validator, types.ValidatorIndex, error) {
	var valIndex types.ValidatorIndex
	if len(validatorId) == params.BeaconConfig().BLSPubkeyLength {
		var ok bool
		valIndex, ok = state.ValidatorIndexByPubkey(bytesutil.ToBytes48(validatorId))
		if !ok {
			return nil, types.ValidatorIndex(0), fmt.Errorf("could not find validator with public key: %#x", validatorId)
		}
	} else {
		index, err := strconv.ParseUint(string(validatorId), 10, 64)
		if err != nil {
			return nil, types.ValidatorIndex(0), errors.Wrap(err, "could not decode validator id")
		}
		valIndex = types.ValidatorIndex(index)
	}
	v1alphaVal, err := state.ValidatorAtIndex(valIndex)
	if err != nil {
		return nil, types.ValidatorIndex(0), err
	}
	return migration.V1Alpha1ValidatorToV1(v1alphaVal), valIndex, nil
}
