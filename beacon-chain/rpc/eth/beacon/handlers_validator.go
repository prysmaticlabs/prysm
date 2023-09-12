package beacon

import (
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
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

// GetValidators returns filterable list of validators with their balance, status and index.
func (s *Server) GetValidators(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetValidators")
	defer span.End()

	var err error
	mv := mux.Vars(r)
	stateId := mv["state_id"]
	if stateId == "" {
		http2.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}

	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}
	ids, ok := decodeIds(w, st, r.URL.Query()["id"])
	if !ok {
		return
	}
	readOnlyVals, ok := valsFromIds(w, st, ids)
	if !ok {
		return
	}
	epoch := slots.ToEpoch(st.Slot())
	allBalances := st.Balances()

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

	// Exit early if no matching validators were found or we don't want to further filter validators by status.
	if len(readOnlyVals) == 0 || len(statuses) == 0 {
		containers := make([]*ValidatorContainer, len(readOnlyVals))
		for i, val := range readOnlyVals {
			valStatus, err := helpers.ValidatorSubStatus(val, epoch)
			if err != nil {
				http2.HandleError(w, "Could not get validator status: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if len(ids) == 0 {
				containers[i] = valContainerFromReadOnlyVal(val, primitives.ValidatorIndex(i), allBalances[i], valStatus)
			} else {
				containers[i] = valContainerFromReadOnlyVal(val, ids[i], allBalances[ids[i]], valStatus)
			}
		}
		resp := &GetValidatorsResponse{
			Data:                containers,
			ExecutionOptimistic: isOptimistic,
			Finalized:           isFinalized,
		}
		http2.WriteJson(w, resp)
		return
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
	valContainers := make([]*ValidatorContainer, 0, len(readOnlyVals))
	for i, val := range readOnlyVals {
		valStatus, err := helpers.ValidatorStatus(val, epoch)
		if err != nil {
			http2.HandleError(w, "Could not get validator status: "+err.Error(), http.StatusInternalServerError)
			return
		}
		valSubStatus, err := helpers.ValidatorSubStatus(val, epoch)
		if err != nil {
			http2.HandleError(w, "Could not get validator status: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if filteredStatuses[valStatus] || filteredStatuses[valSubStatus] {
			var container *ValidatorContainer
			if len(ids) == 0 {
				container = valContainerFromReadOnlyVal(val, primitives.ValidatorIndex(i), allBalances[i], valSubStatus)
			} else {
				container = valContainerFromReadOnlyVal(val, ids[i], allBalances[ids[i]], valSubStatus)
			}
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
/*func (bs *Server) GetValidator(ctx context.Context, req *ethpb.StateValidatorRequest) (*ethpb.StateValidatorResponse, error) {
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
}*/

// GetValidatorBalances returns a filterable list of validator balances.
/*func (bs *Server) GetValidatorBalances(ctx context.Context, req *ethpb.ValidatorBalancesRequest) (*ethpb.ValidatorBalancesResponse, error) {
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
}*/

func decodeIds(w http.ResponseWriter, st state.BeaconState, rawIds []string) ([]primitives.ValidatorIndex, bool) {
	ids := make([]primitives.ValidatorIndex, 0, len(rawIds))
	numVals := uint64(st.NumValidators())
	for _, rawId := range rawIds {
		pubkey, err := hexutil.Decode(rawId)
		if err == nil {
			if len(pubkey) != fieldparams.BLSPubkeyLength {
				http2.HandleError(w, fmt.Sprintf("Pubkey length is %d instead of %d", len(pubkey), fieldparams.BLSPubkeyLength), http.StatusBadRequest)
				return nil, false
			}
			valIndex, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubkey))
			if !ok {
				// Ignore well-formed yet unknown public keys.
				continue
			}
			ids = append(ids, valIndex)
			continue
		}

		index, err := strconv.ParseUint(rawId, 10, 64)
		if err != nil {
			http2.HandleError(w, fmt.Sprintf("Invalid validator ID %s", rawId), http.StatusBadRequest)
			return nil, false
		}
		if index >= numVals {
			// Ignore well-formed yet unknown public keys.
			continue
		}
		ids = append(ids, primitives.ValidatorIndex(index))
	}
	return ids, true
}

func valsFromIds(w http.ResponseWriter, st state.BeaconState, ids []primitives.ValidatorIndex) ([]state.ReadOnlyValidator, bool) {
	var vals []state.ReadOnlyValidator
	if len(ids) == 0 {
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
		vals = make([]state.ReadOnlyValidator, 0, len(ids))
		for _, id := range ids {
			val, err := st.ValidatorAtIndex(id)
			if err != nil {
				http2.HandleError(w, fmt.Sprintf("Could not get validator at index %d: %s", id, err.Error()), http.StatusInternalServerError)
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

func valContainerFromReadOnlyVal(
	val state.ReadOnlyValidator,
	id primitives.ValidatorIndex,
	bal uint64,
	valStatus validator.ValidatorStatus,
) *ValidatorContainer {
	pubkey := val.PublicKey()
	return &ValidatorContainer{
		Index:   strconv.FormatUint(uint64(id), 10),
		Balance: strconv.FormatUint(bal, 10),
		Status:  valStatus.String(),
		Validator: &Validator{
			Pubkey:                     hexutil.Encode(pubkey[:]),
			WithdrawalCredentials:      hexutil.Encode(val.WithdrawalCredentials()),
			EffectiveBalance:           strconv.FormatUint(val.EffectiveBalance(), 10),
			Slashed:                    val.Slashed(),
			ActivationEligibilityEpoch: strconv.FormatUint(uint64(val.ActivationEligibilityEpoch()), 10),
			ActivationEpoch:            strconv.FormatUint(uint64(val.ActivationEpoch()), 10),
			ExitEpoch:                  strconv.FormatUint(uint64(val.ExitEpoch()), 10),
			WithdrawableEpoch:          strconv.FormatUint(uint64(val.WithdrawableEpoch()), 10),
		},
	}
}
