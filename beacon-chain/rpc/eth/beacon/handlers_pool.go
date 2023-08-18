package beacon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/operation"
	corehelpers "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

// ListAttestations retrieves attestations known by the node but
// not necessarily incorporated into any block. Allows filtering by committee index or slot.
func (s *Server) ListAttestations(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "beacon.ListAttestations")
	defer span.End()

	ok, rawSlot, slot := shared.UintFromQuery(w, r, "slot")
	if !ok {
		return
	}
	ok, rawCommitteeIndex, committeeIndex := shared.UintFromQuery(w, r, "committee_index")
	if !ok {
		return
	}

	attestations := s.AttestationsPool.AggregatedAttestations()
	unaggAtts, err := s.AttestationsPool.UnaggregatedAttestations()
	if err != nil {
		http2.HandleError(w, "Could not get unaggregated attestations: "+err.Error(), http.StatusInternalServerError)
		return
	}
	attestations = append(attestations, unaggAtts...)
	isEmptyReq := rawSlot == "" && rawCommitteeIndex == ""
	if isEmptyReq {
		allAtts := make([]*shared.Attestation, len(attestations))
		for i, att := range attestations {
			allAtts[i] = shared.AttestationFromConsensus(att)
		}
		http2.WriteJson(w, &ListAttestationsResponse{Data: allAtts})
		return
	}

	bothDefined := rawSlot != "" && rawCommitteeIndex != ""
	filteredAtts := make([]*shared.Attestation, 0, len(attestations))
	for _, att := range attestations {
		committeeIndexMatch := rawCommitteeIndex != "" && att.Data.CommitteeIndex == primitives.CommitteeIndex(committeeIndex)
		slotMatch := rawSlot != "" && att.Data.Slot == primitives.Slot(slot)
		shouldAppend := (bothDefined && committeeIndexMatch && slotMatch) || (!bothDefined && (committeeIndexMatch || slotMatch))
		if shouldAppend {
			filteredAtts = append(filteredAtts, shared.AttestationFromConsensus(att))
		}
	}
	http2.WriteJson(w, &ListAttestationsResponse{Data: filteredAtts})
}

// SubmitAttestations submits an Attestation object to node. If the attestation passes all validation
// constraints, node MUST publish the attestation on an appropriate subnet.
func (s *Server) SubmitAttestations(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.SubmitAttestations")
	defer span.End()

	if r.Body == http.NoBody {
		http2.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	}
	var req SubmitAttestationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req.Data); err != nil {
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

	var validAttestations []*ethpbalpha.Attestation
	var attFailures []*shared.IndexedVerificationFailure
	for i, sourceAtt := range req.Data {
		att, err := sourceAtt.ToConsensus()
		if err != nil {
			attFailures = append(attFailures, &shared.IndexedVerificationFailure{
				Index:   i,
				Message: "Could not convert request attestation to consensus attestation: " + err.Error(),
			})
			continue
		}
		if _, err = bls.SignatureFromBytes(att.Signature); err != nil {
			attFailures = append(attFailures, &shared.IndexedVerificationFailure{
				Index:   i,
				Message: "Incorrect attestation signature: " + err.Error(),
			})
			continue
		}

		// Broadcast the unaggregated attestation on a feed to notify other services in the beacon node
		// of a received unaggregated attestation.
		// Note we can't send for aggregated att because we don't have selection proof.
		if !corehelpers.IsAggregated(att) {
			s.OperationNotifier.OperationFeed().Send(&feed.Event{
				Type: operation.UnaggregatedAttReceived,
				Data: &operation.UnAggregatedAttReceivedData{
					Attestation: att,
				},
			})
		}

		validAttestations = append(validAttestations, att)
	}

	failedBroadcasts := make([]string, 0)
	for i, att := range validAttestations {
		// Determine subnet to broadcast attestation to
		wantedEpoch := slots.ToEpoch(att.Data.Slot)
		vals, err := s.HeadFetcher.HeadValidatorsIndices(ctx, wantedEpoch)
		if err != nil {
			http2.HandleError(w, "Could not get head validator indices: "+err.Error(), http.StatusInternalServerError)
			return
		}
		subnet := corehelpers.ComputeSubnetFromCommitteeAndSlot(uint64(len(vals)), att.Data.CommitteeIndex, att.Data.Slot)

		if err = s.Broadcaster.BroadcastAttestation(ctx, subnet, att); err != nil {
			failedBroadcasts = append(failedBroadcasts, strconv.Itoa(i))
			log.WithError(err).Errorf("could not broadcast attestation at index %d", i)
		}

		if corehelpers.IsAggregated(att) {
			if err = s.AttestationsPool.SaveAggregatedAttestation(att); err != nil {
				log.WithError(err).Error("could not save aggregated att")
			}
		} else {
			if err = s.AttestationsPool.SaveUnaggregatedAttestation(att); err != nil {
				log.WithError(err).Error("could not save unaggregated att")
			}
		}
	}
	if len(failedBroadcasts) > 0 {
		http2.HandleError(
			w,
			fmt.Sprintf("Attestations at index %s could not be broadcasted", strings.Join(failedBroadcasts, ", ")),
			http.StatusInternalServerError,
		)
		return
	}

	if len(attFailures) > 0 {
		failuresErr := &shared.IndexedVerificationFailureError{
			Code:     http.StatusBadRequest,
			Message:  "One or more attestations failed validation",
			Failures: attFailures,
		}
		http2.WriteError(w, failuresErr)
	}
}
