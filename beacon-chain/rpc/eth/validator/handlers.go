package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-playground/validator/v10"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/builder"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/core"
	rpchelpers "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	validator2 "github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

// GetAggregateAttestation aggregates all attestations matching the given attestation data root and slot, returning the aggregated result.
func (s *Server) GetAggregateAttestation(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.GetAggregateAttestation")
	defer span.End()

	attDataRoot := r.URL.Query().Get("attestation_data_root")
	valid := shared.ValidateHex(w, "Attestation data root", attDataRoot)
	if !valid {
		return
	}
	rawSlot := r.URL.Query().Get("slot")
	slot, valid := shared.ValidateUint(w, "Slot", rawSlot)
	if !valid {
		return
	}

	if err := s.AttestationsPool.AggregateUnaggregatedAttestations(ctx); err != nil {
		http2.HandleError(w, "Could not aggregate unaggregated attestations: "+err.Error(), http.StatusBadRequest)
		return
	}

	allAtts := s.AttestationsPool.AggregatedAttestations()
	var bestMatchingAtt *ethpbalpha.Attestation
	for _, att := range allAtts {
		if att.Data.Slot == primitives.Slot(slot) {
			root, err := att.Data.HashTreeRoot()
			if err != nil {
				http2.HandleError(w, "Could not get attestation data root: "+err.Error(), http.StatusInternalServerError)
				return
			}
			attDataRootBytes, err := hexutil.Decode(attDataRoot)
			if err != nil {
				http2.HandleError(w, "Could not decode attestation data root into bytes: "+err.Error(), http.StatusBadRequest)
				return
			}
			if bytes.Equal(root[:], attDataRootBytes) {
				if bestMatchingAtt == nil || len(att.AggregationBits) > len(bestMatchingAtt.AggregationBits) {
					bestMatchingAtt = att
				}
			}
		}
	}
	if bestMatchingAtt == nil {
		http2.HandleError(w, "No matching attestation found", http.StatusNotFound)
		return
	}

	response := &AggregateAttestationResponse{
		Data: &shared.Attestation{
			AggregationBits: hexutil.Encode(bestMatchingAtt.AggregationBits),
			Data: &shared.AttestationData{
				Slot:            strconv.FormatUint(uint64(bestMatchingAtt.Data.Slot), 10),
				CommitteeIndex:  strconv.FormatUint(uint64(bestMatchingAtt.Data.CommitteeIndex), 10),
				BeaconBlockRoot: hexutil.Encode(bestMatchingAtt.Data.BeaconBlockRoot),
				Source: &shared.Checkpoint{
					Epoch: strconv.FormatUint(uint64(bestMatchingAtt.Data.Source.Epoch), 10),
					Root:  hexutil.Encode(bestMatchingAtt.Data.Source.Root),
				},
				Target: &shared.Checkpoint{
					Epoch: strconv.FormatUint(uint64(bestMatchingAtt.Data.Target.Epoch), 10),
					Root:  hexutil.Encode(bestMatchingAtt.Data.Target.Root),
				},
			},
			Signature: hexutil.Encode(bestMatchingAtt.Signature),
		}}
	http2.WriteJson(w, response)
}

// SubmitContributionAndProofs publishes multiple signed sync committee contribution and proofs.
func (s *Server) SubmitContributionAndProofs(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.SubmitContributionAndProofs")
	defer span.End()

	var req SubmitContributionAndProofsRequest
	err := json.NewDecoder(r.Body).Decode(&req.Data)
	switch {
	case err == io.EOF:
		http2.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		http2.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Data) == 0 {
		http2.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	}
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		http2.HandleError(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, item := range req.Data {
		consensusItem, err := item.ToConsensus()
		if err != nil {
			http2.HandleError(w, "Could not convert request contribution to consensus contribution: "+err.Error(), http.StatusBadRequest)
			return
		}
		rpcError := s.CoreService.SubmitSignedContributionAndProof(ctx, consensusItem)
		if rpcError != nil {
			http2.HandleError(w, rpcError.Err.Error(), core.ErrorReasonToHTTP(rpcError.Reason))
		}
	}
}

// SubmitAggregateAndProofs verifies given aggregate and proofs and publishes them on appropriate gossipsub topic.
func (s *Server) SubmitAggregateAndProofs(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.SubmitAggregateAndProofs")
	defer span.End()

	var req SubmitAggregateAndProofsRequest
	err := json.NewDecoder(r.Body).Decode(&req.Data)
	switch {
	case err == io.EOF:
		http2.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		http2.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Data) == 0 {
		http2.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	}
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		http2.HandleError(w, err.Error(), http.StatusBadRequest)
		return
	}

	broadcastFailed := false
	for _, item := range req.Data {
		consensusItem, err := item.ToConsensus()
		if err != nil {
			http2.HandleError(w, "Could not convert request aggregate to consensus aggregate: "+err.Error(), http.StatusBadRequest)
			return
		}
		rpcError := s.CoreService.SubmitSignedAggregateSelectionProof(
			ctx,
			&ethpbalpha.SignedAggregateSubmitRequest{SignedAggregateAndProof: consensusItem},
		)
		if rpcError != nil {
			_, ok := rpcError.Err.(*core.AggregateBroadcastFailedError)
			if ok {
				broadcastFailed = true
			} else {
				http2.HandleError(w, rpcError.Err.Error(), core.ErrorReasonToHTTP(rpcError.Reason))
				return
			}
		}
	}

	if broadcastFailed {
		http2.HandleError(w, "Could not broadcast one or more signed aggregated attestations", http.StatusInternalServerError)
	}
}

// SubmitSyncCommitteeSubscription subscribe to a number of sync committee subnets.
//
// Subscribing to sync committee subnets is an action performed by VC to enable
// network participation, and only required if the VC has an active
// validator in an active sync committee.
func (s *Server) SubmitSyncCommitteeSubscription(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.SubmitSyncCommitteeSubscription")
	defer span.End()

	if shared.IsSyncing(ctx, w, s.SyncChecker, s.HeadFetcher, s.TimeFetcher, s.OptimisticModeFetcher) {
		return
	}

	var req SubmitSyncCommitteeSubscriptionsRequest
	err := json.NewDecoder(r.Body).Decode(&req.Data)
	switch {
	case err == io.EOF:
		http2.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		http2.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Data) == 0 {
		http2.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	}
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		http2.HandleError(w, err.Error(), http.StatusBadRequest)
		return
	}

	st, err := s.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		http2.HandleError(w, "Could not get head state: "+err.Error(), http.StatusInternalServerError)
		return
	}
	currEpoch := slots.ToEpoch(st.Slot())
	validators := make([]state.ReadOnlyValidator, len(req.Data))
	subscriptions := make([]*validator2.SyncCommitteeSubscription, len(req.Data))
	for i, item := range req.Data {
		consensusItem, err := item.ToConsensus()
		if err != nil {
			http2.HandleError(w, "Could not convert request subscription to consensus subscription: "+err.Error(), http.StatusBadRequest)
			return
		}
		subscriptions[i] = consensusItem
		val, err := st.ValidatorAtIndexReadOnly(consensusItem.ValidatorIndex)
		if err != nil {
			http2.HandleError(
				w,
				fmt.Sprintf("Could not get validator at index %d: %s", consensusItem.ValidatorIndex, err.Error()),
				http.StatusInternalServerError,
			)
			return
		}
		valStatus, err := rpchelpers.ValidatorSubStatus(val, currEpoch)
		if err != nil {
			http2.HandleError(
				w,
				fmt.Sprintf("Could not get validator status at index %d: %s", consensusItem.ValidatorIndex, err.Error()),
				http.StatusInternalServerError,
			)
			return
		}
		if valStatus != validator2.ActiveOngoing && valStatus != validator2.ActiveExiting {
			http2.HandleError(
				w,
				fmt.Sprintf("Validator at index %d is not active or exiting", consensusItem.ValidatorIndex),
				http.StatusBadRequest,
			)
			return
		}
		validators[i] = val
	}

	startEpoch, err := slots.SyncCommitteePeriodStartEpoch(currEpoch)
	if err != nil {
		http2.HandleError(w, "Could not get sync committee period start epoch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for i, sub := range subscriptions {
		if sub.UntilEpoch <= currEpoch {
			http2.HandleError(
				w,
				fmt.Sprintf("Epoch for subscription at index %d is in the past. It must be at least %d", i, currEpoch+1),
				http.StatusBadRequest,
			)
			return
		}
		maxValidUntilEpoch := startEpoch + params.BeaconConfig().EpochsPerSyncCommitteePeriod*2
		if sub.UntilEpoch > maxValidUntilEpoch {
			http2.HandleError(
				w,
				fmt.Sprintf("Epoch for subscription at index %d is too far in the future. It can be at most %d", i, maxValidUntilEpoch),
				http.StatusBadRequest,
			)
			return
		}
	}

	for i, sub := range subscriptions {
		pubkey48 := validators[i].PublicKey()
		// Handle overflow in the event current epoch is less than end epoch.
		// This is an impossible condition, so it is a defensive check.
		epochsToWatch, err := sub.UntilEpoch.SafeSub(uint64(startEpoch))
		if err != nil {
			epochsToWatch = 0
		}
		epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot)) * time.Second
		totalDuration := epochDuration * time.Duration(epochsToWatch)

		cache.SyncSubnetIDs.AddSyncCommitteeSubnets(pubkey48[:], startEpoch, sub.SyncCommitteeIndices, totalDuration)
	}
}

// SubmitBeaconCommitteeSubscription searches using discv5 for peers related to the provided subnet information
// and replaces current peers with those ones if necessary.
func (s *Server) SubmitBeaconCommitteeSubscription(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.SubmitBeaconCommitteeSubscription")
	defer span.End()

	if shared.IsSyncing(ctx, w, s.SyncChecker, s.HeadFetcher, s.TimeFetcher, s.OptimisticModeFetcher) {
		return
	}

	var req SubmitBeaconCommitteeSubscriptionsRequest
	err := json.NewDecoder(r.Body).Decode(&req.Data)
	switch {
	case err == io.EOF:
		http2.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		http2.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Data) == 0 {
		http2.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	}
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		http2.HandleError(w, err.Error(), http.StatusBadRequest)
		return
	}

	st, err := s.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		http2.HandleError(w, "Could not get head state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Verify validators at the beginning to return early if request is invalid.
	validators := make([]state.ReadOnlyValidator, len(req.Data))
	subscriptions := make([]*validator2.BeaconCommitteeSubscription, len(req.Data))
	for i, item := range req.Data {
		consensusItem, err := item.ToConsensus()
		if err != nil {
			http2.HandleError(w, "Could not convert request subscription to consensus subscription: "+err.Error(), http.StatusBadRequest)
			return
		}
		subscriptions[i] = consensusItem
		val, err := st.ValidatorAtIndexReadOnly(consensusItem.ValidatorIndex)
		if err != nil {
			if outOfRangeErr, ok := err.(*state_native.ValidatorIndexOutOfRangeError); ok {
				http2.HandleError(w, "Could not get validator: "+outOfRangeErr.Error(), http.StatusBadRequest)
				return
			}
			http2.HandleError(w, "Could not get validator: "+err.Error(), http.StatusInternalServerError)
			return
		}
		validators[i] = val
	}

	fetchValsLen := func(slot primitives.Slot) (uint64, error) {
		wantedEpoch := slots.ToEpoch(slot)
		vals, err := s.HeadFetcher.HeadValidatorsIndices(ctx, wantedEpoch)
		if err != nil {
			return 0, err
		}
		return uint64(len(vals)), nil
	}

	// Request the head validator indices of epoch represented by the first requested slot.
	currValsLen, err := fetchValsLen(subscriptions[0].Slot)
	if err != nil {
		http2.HandleError(w, "Could not retrieve head validator length: "+err.Error(), http.StatusInternalServerError)
		return
	}
	currEpoch := slots.ToEpoch(subscriptions[0].Slot)
	for _, sub := range subscriptions {
		// If epoch has changed, re-request active validators length
		if currEpoch != slots.ToEpoch(sub.Slot) {
			currValsLen, err = fetchValsLen(sub.Slot)
			if err != nil {
				http2.HandleError(w, "Could not retrieve head validator length: "+err.Error(), http.StatusInternalServerError)
				return
			}
			currEpoch = slots.ToEpoch(sub.Slot)
		}
		subnet := helpers.ComputeSubnetFromCommitteeAndSlot(currValsLen, sub.CommitteeIndex, sub.Slot)
		cache.SubnetIDs.AddAttesterSubnetID(sub.Slot, subnet)
		if sub.IsAggregator {
			cache.SubnetIDs.AddAggregatorSubnetID(sub.Slot, subnet)
		}
	}
	for _, val := range validators {
		valStatus, err := rpchelpers.ValidatorStatus(val, currEpoch)
		if err != nil {
			http2.HandleError(w, "Could not retrieve validator status: "+err.Error(), http.StatusInternalServerError)
			return
		}
		pubkey := val.PublicKey()
		core.AssignValidatorToSubnet(pubkey[:], valStatus)
	}
}

// GetAttestationData requests that the beacon node produces attestation data for
// the requested committee index and slot based on the nodes current head.
func (s *Server) GetAttestationData(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.GetAttestationData")
	defer span.End()

	if shared.IsSyncing(ctx, w, s.SyncChecker, s.HeadFetcher, s.TimeFetcher, s.OptimisticModeFetcher) {
		return
	}

	if isOptimistic, err := shared.IsOptimistic(ctx, w, s.OptimisticModeFetcher); isOptimistic || err != nil {
		return
	}

	rawSlot := r.URL.Query().Get("slot")
	slot, valid := shared.ValidateUint(w, "Slot", rawSlot)
	if !valid {
		return
	}
	rawCommitteeIndex := r.URL.Query().Get("committee_index")
	committeeIndex, valid := shared.ValidateUint(w, "Committee Index", rawCommitteeIndex)
	if !valid {
		return
	}

	attestationData, rpcError := s.CoreService.GetAttestationData(ctx, &ethpbalpha.AttestationDataRequest{
		Slot:           primitives.Slot(slot),
		CommitteeIndex: primitives.CommitteeIndex(committeeIndex),
	})

	if rpcError != nil {
		http2.HandleError(w, rpcError.Err.Error(), core.ErrorReasonToHTTP(rpcError.Reason))
		return
	}

	response := &GetAttestationDataResponse{
		Data: &shared.AttestationData{
			Slot:            strconv.FormatUint(uint64(attestationData.Slot), 10),
			CommitteeIndex:  strconv.FormatUint(uint64(attestationData.CommitteeIndex), 10),
			BeaconBlockRoot: hexutil.Encode(attestationData.BeaconBlockRoot),
			Source: &shared.Checkpoint{
				Epoch: strconv.FormatUint(uint64(attestationData.Source.Epoch), 10),
				Root:  hexutil.Encode(attestationData.Source.Root),
			},
			Target: &shared.Checkpoint{
				Epoch: strconv.FormatUint(uint64(attestationData.Target.Epoch), 10),
				Root:  hexutil.Encode(attestationData.Target.Root),
			},
		},
	}
	http2.WriteJson(w, response)
}

// ProduceSyncCommitteeContribution requests that the beacon node produce a sync committee contribution.
func (s *Server) ProduceSyncCommitteeContribution(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.ProduceSyncCommitteeContribution")
	defer span.End()

	subIndex := r.URL.Query().Get("subcommittee_index")
	index, valid := shared.ValidateUint(w, "Subcommittee Index", subIndex)
	if !valid {
		return
	}
	rawSlot := r.URL.Query().Get("slot")
	slot, valid := shared.ValidateUint(w, "Slot", rawSlot)
	if !valid {
		return
	}
	rawBlockRoot := r.URL.Query().Get("beacon_block_root")
	blockRoot, err := hexutil.Decode(rawBlockRoot)
	if err != nil {
		http2.HandleError(w, "Invalid Beacon Block Root: "+err.Error(), http.StatusBadRequest)
		return
	}
	contribution, ok := s.produceSyncCommitteeContribution(ctx, w, primitives.Slot(slot), index, []byte(blockRoot))
	if !ok {
		return
	}
	response := &ProduceSyncCommitteeContributionResponse{
		Data: contribution,
	}
	http2.WriteJson(w, response)
}

// ProduceSyncCommitteeContribution requests that the beacon node produce a sync committee contribution.
func (s *Server) produceSyncCommitteeContribution(
	ctx context.Context,
	w http.ResponseWriter,
	slot primitives.Slot,
	index uint64,
	blockRoot []byte,
) (*shared.SyncCommitteeContribution, bool) {
	msgs, err := s.SyncCommitteePool.SyncCommitteeMessages(slot)
	if err != nil {
		http2.HandleError(w, "Could not get sync subcommittee messages: "+err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	if len(msgs) == 0 {
		http2.HandleError(w, "No subcommittee messages found", http.StatusNotFound)
		return nil, false
	}
	sig, aggregatedBits, err := s.CoreService.AggregatedSigAndAggregationBits(
		ctx,
		&ethpbalpha.AggregatedSigAndAggregationBitsRequest{
			Msgs:      msgs,
			Slot:      slot,
			SubnetId:  index,
			BlockRoot: blockRoot,
		},
	)
	if err != nil {
		http2.HandleError(w, "Could not get contribution data: "+err.Error(), http.StatusInternalServerError)
		return nil, false
	}

	return &shared.SyncCommitteeContribution{
		Slot:              strconv.FormatUint(uint64(slot), 10),
		BeaconBlockRoot:   hexutil.Encode(blockRoot),
		SubcommitteeIndex: strconv.FormatUint(index, 10),
		AggregationBits:   hexutil.Encode(aggregatedBits),
		Signature:         hexutil.Encode(sig),
	}, true
}

// RegisterValidator requests that the beacon node stores valid validator registrations and calls the builder apis to update the custom builder
func (s *Server) RegisterValidator(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.RegisterValidators")
	defer span.End()

	if s.BlockBuilder == nil || !s.BlockBuilder.Configured() {
		http2.HandleError(w, fmt.Sprintf("Could not register block builder: %v", builder.ErrNoBuilder), http.StatusBadRequest)
		return
	}

	var jsonRegistrations []*shared.SignedValidatorRegistration
	err := json.NewDecoder(r.Body).Decode(&jsonRegistrations)
	switch {
	case err == io.EOF:
		http2.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		http2.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	validate := validator.New()
	registrations := make([]*ethpbalpha.SignedValidatorRegistrationV1, len(jsonRegistrations))
	for i, registration := range jsonRegistrations {
		if err := validate.Struct(registration); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		reg, err := registration.ToConsensus()
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}

		registrations[i] = reg
	}
	if len(registrations) == 0 {
		http2.HandleError(w, "Validator registration request is empty", http.StatusBadRequest)
		return
	}
	if err := s.BlockBuilder.RegisterValidator(ctx, registrations); err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
