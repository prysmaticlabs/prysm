package beacon

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	statenative "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetValidators returns filterable list of validators with their balance, status and index.
func (s *Server) GetValidators(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetValidators")
	defer span.End()

	var err error
	stateId := mux.Vars(r)["state_id"]
	if stateId == "" {
		http2.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}

	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}
	valContainers, ok := valContainersFromIds(w, st, r.URL.Query()["id"])
	if !ok {
		return
	}

	statuses := r.URL.Query()["status"]

	isOptimistic, err := s.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		http2.HandleError(w, "Could not check optimistic status: "+err.Error(), http.StatusInternalServerError)
		return
	}
	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		http2.HandleError(w, "Could not calculate root of latest block header: "+err.Error(), http.StatusInternalServerError)
		return
	}
	isFinalized := s.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	// Exit early if no matching validators we found or we don't want to further filter validators by status.
	if len(valContainers) == 0 || len(statuses) == 0 {
		resp := &GetValidatorsResponse{
			Data:                valContainers,
			ExecutionOptimistic: isOptimistic,
			Finalized:           isFinalized,
		}
		http2.WriteJson(w, resp)
	}

	filterStatus := make(map[validator.ValidatorStatus]bool, len(statuses))
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

	resp := &GetValidatorsResponse{
		Data:                filteredVals,
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
	}
	http2.WriteJson(w, resp)
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
	valContainer, err := valContainersFromIds(st, [][]byte{req.ValidatorId})
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

// GetValidatorBalances returns a filterable list of validator balances.
func (bs *Server) GetValidatorBalances(ctx context.Context, req *ethpb.ValidatorBalancesRequest) (*ethpb.ValidatorBalancesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListValidatorBalances")
	defer span.End()

	st, err := bs.Stater.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	valContainers, err := valContainersFromIds(st, req.Id)
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

// This function returns a list of validator containers based on IDs. A validator ID can be its public key or its index.
func valContainersFromIds(w http.ResponseWriter, state state.BeaconState, validatorIds []string) ([]*ValidatorContainer, bool) {
	epoch := slots.ToEpoch(state.Slot())
	var valContainers []*ValidatorContainer
	allBalances := state.Balances()
	if len(validatorIds) == 0 {
		allValidators := state.Validators()
		valContainers = make([]*ValidatorContainer, len(allValidators))
		for i, val := range allValidators {
			readOnlyVal, err := statenative.NewValidator(val)
			if err != nil {
				http2.HandleError(w, "Could not convert validator: "+err.Error(), http.StatusInternalServerError)
				return nil, false
			}
			subStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
			if err != nil {
				http2.HandleError(w, "Could not get validator sub status: "+err.Error(), http.StatusInternalServerError)
				return nil, false
			}
			valContainers[i] = &ValidatorContainer{
				Index:   strconv.Itoa(i),
				Balance: strconv.FormatUint(allBalances[i], 10),
				Status:  subStatus.String(),
				Validator: &Validator{
					Pubkey:                     hexutil.Encode(val.PublicKey),
					WithdrawalCredentials:      hexutil.Encode(val.WithdrawalCredentials),
					EffectiveBalance:           strconv.FormatUint(val.EffectiveBalance, 10),
					Slashed:                    val.Slashed,
					ActivationEligibilityEpoch: strconv.FormatUint(uint64(val.ActivationEligibilityEpoch), 10),
					ActivationEpoch:            strconv.FormatUint(uint64(val.ActivationEpoch), 10),
					ExitEpoch:                  strconv.FormatUint(uint64(val.ExitEpoch), 10),
					WithdrawableEpoch:          strconv.FormatUint(uint64(val.WithdrawableEpoch), 10),
				},
			}
		}
	} else {
		valContainers = make([]*ValidatorContainer, 0, len(validatorIds))
		for _, validatorId := range validatorIds {
			var valIndex primitives.ValidatorIndex
			pubkey, err := hexutil.Decode(validatorId)
			if err == nil {
				if len(pubkey) == fieldparams.BLSPubkeyLength {
					var ok bool
					valIndex, ok = state.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubkey))
					if !ok {
						// Ignore well-formed yet unknown public keys.
						continue
					}
				}
				http2.HandleError(w, fmt.Sprintf("Pubkey length is %d instead of %d", len(pubkey), fieldparams.BLSPubkeyLength), http.StatusBadRequest)
				return nil, false
			}

			index, err := strconv.ParseUint(validatorId, 10, 64)
			if err != nil {
				http2.HandleError(w, fmt.Sprintf("Invalid validator ID %s", validatorId), http.StatusBadRequest)
				return nil, false
			}
			valIndex = primitives.ValidatorIndex(index)
			val, err := state.ValidatorAtIndex(valIndex)
			if err != nil {
				if _, ok := err.(*statenative.ValidatorIndexOutOfRangeError); ok {
					// Ignore well-formed yet unknown indexes.
					continue
				}
				http2.HandleError(w, fmt.Sprintf("Could not get validator at index %d: %s", valIndex, err.Error()), http.StatusInternalServerError)
				return nil, false
			}

			readOnlyVal, err := statenative.NewValidator(val)
			if err != nil {
				http2.HandleError(w, "Could not convert validator: "+err.Error(), http.StatusInternalServerError)
				return nil, false
			}
			subStatus, err := helpers.ValidatorSubStatus(readOnlyVal, epoch)
			if err != nil {
				http2.HandleError(w, "Could not get validator sub status: "+err.Error(), http.StatusInternalServerError)
				return nil, false
			}
			valContainers = append(valContainers, &ValidatorContainer{
				Index:   strconv.FormatUint(uint64(valIndex), 10),
				Balance: strconv.FormatUint(allBalances[valIndex], 10),
				Status:  subStatus.String(),
				Validator: &Validator{
					Pubkey:                     hexutil.Encode(val.PublicKey),
					WithdrawalCredentials:      hexutil.Encode(val.WithdrawalCredentials),
					EffectiveBalance:           strconv.FormatUint(val.EffectiveBalance, 10),
					Slashed:                    val.Slashed,
					ActivationEligibilityEpoch: strconv.FormatUint(uint64(val.ActivationEligibilityEpoch), 10),
					ActivationEpoch:            strconv.FormatUint(uint64(val.ActivationEpoch), 10),
					ExitEpoch:                  strconv.FormatUint(uint64(val.ExitEpoch), 10),
					WithdrawableEpoch:          strconv.FormatUint(uint64(val.WithdrawableEpoch), 10),
				},
			})
		}
	}

	return valContainers, true
}
