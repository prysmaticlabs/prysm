package beacon

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	readOnlyVals, ok := valsFromIds(w, st, r.URL.Query()["id"])
	if !ok {
		return
	}

	statuses := r.URL.Query()["status"]
	for i, ss := range statuses {
		statuses[i] = strings.ToLower(ss)
	}

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
	if len(readOnlyVals) == 0 || len(statuses) == 0 {

		resp := &GetValidatorsResponse{
			Data:                readOnlyVals,
			ExecutionOptimistic: isOptimistic,
			Finalized:           isFinalized,
		}
		http2.WriteJson(w, resp)
	}

	filteredStatuses := make(map[validator.ValidatorStatus]bool, len(statuses))
	for _, ss := range statuses {
		ok, vs := validator.ValidatorStatusFromString(ss)
		if !ok {
			http2.HandleError(w, "Invalid status "+ss, http.StatusBadRequest)
			return
		}
		filteredStatuses[vs] = true
	}
	epoch := slots.ToEpoch(st.Slot())
	valContainers := make([]*ValidatorContainer, 0, len(readOnlyVals))
	for _, val := range readOnlyVals {
		valStatus, err := helpers.ValidatorStatus(val, epoch)
		if err != nil {
			http2.HandleError(w, "Could not get validator status: "+err.Error(), http.StatusInternalServerError)
			return
		}
		valSubStatus, err := helpers.ValidatorSubStatus(val, epoch)
		if err != nil {
			http2.HandleError(w, "Could not get validator sub status: "+err.Error(), http.StatusInternalServerError)
			return
		}
		container := &ValidatorContainer{
			Index:     "",
			Balance:   "",
			Status:    "",
			Validator: nil,
		}
		if filteredStatuses[valStatus] || filteredStatuses[valSubStatus] {
			valContainers = append(valContainers, container)
		}
	}

	resp := &GetValidatorsResponse{
		Data:                valContainers,
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

// This function returns a list of read-only validators based on IDs. A validator ID can be its public key or its index.
func valsFromIds(w http.ResponseWriter, st state.BeaconState, valIds []string) ([]state.ReadOnlyValidator, bool) {
	var vals []state.ReadOnlyValidator
	if len(valIds) == 0 {
		allVals := st.Validators()
		vals = make([]state.ReadOnlyValidator, len(allVals))
		for i, val := range allVals {
			readOnlyVal, err := statenative.NewValidator(val)
			if err != nil {
				http2.HandleError(w, "Could not convert validator: "+err.Error(), http.StatusInternalServerError)
				return nil, false
			}
			vals[i] = readOnlyVal
		}
	} else {
		vals = make([]state.ReadOnlyValidator, 0, len(valIds))
		for _, valId := range valIds {
			var valIndex primitives.ValidatorIndex
			pubkey, err := hexutil.Decode(valId)
			if err == nil {
				if len(pubkey) == fieldparams.BLSPubkeyLength {
					var ok bool
					valIndex, ok = st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubkey))
					if !ok {
						// Ignore well-formed yet unknown public keys.
						continue
					}
				}
				http2.HandleError(w, fmt.Sprintf("Pubkey length is %d instead of %d", len(pubkey), fieldparams.BLSPubkeyLength), http.StatusBadRequest)
				return nil, false
			}

			index, err := strconv.ParseUint(valId, 10, 64)
			if err != nil {
				http2.HandleError(w, fmt.Sprintf("Invalid validator ID %s", valId), http.StatusBadRequest)
				return nil, false
			}
			valIndex = primitives.ValidatorIndex(index)
			val, err := st.ValidatorAtIndex(valIndex)
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
			vals = append(vals, readOnlyVal)
		}
	}

	return vals, true
}
