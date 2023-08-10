package beacon

import (
	"net/http"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	"go.opencensus.io/trace"
)

// ListAttestations retrieves attestations known by the node but
// not necessarily incorporated into any block. Allows filtering by committee index or slot.
func (s *Server) ListAttestations(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "beacon.ListAttestations")
	defer span.End()

	rawSlot := r.URL.Query().Get("slot")
	var slot uint64
	if rawSlot != "" {
		var valid bool
		slot, valid = shared.ValidateUint(w, "Slot", rawSlot)
		if !valid {
			return
		}
	}
	rawCommitteeIndex := r.URL.Query().Get("committee_index")
	var committeeIndex uint64
	if rawCommitteeIndex != "" {
		var valid bool
		committeeIndex, valid = shared.ValidateUint(w, "Committee index", rawCommitteeIndex)
		if !valid {
			return
		}
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

	filteredAtts := make([]*shared.Attestation, 0, len(attestations))
	for _, att := range attestations {
		bothDefined := rawSlot != "" && rawCommitteeIndex != ""
		committeeIndexMatch := rawCommitteeIndex != "" && att.Data.CommitteeIndex == primitives.CommitteeIndex(committeeIndex)
		slotMatch := rawSlot != "" && att.Data.Slot == primitives.Slot(slot)

		if bothDefined && committeeIndexMatch && slotMatch {
			filteredAtts = append(filteredAtts, shared.AttestationFromConsensus(att))
		} else if !bothDefined && (committeeIndexMatch || slotMatch) {
			filteredAtts = append(filteredAtts, shared.AttestationFromConsensus(att))
		}
	}
	http2.WriteJson(w, &ListAttestationsResponse{Data: filteredAtts})
	return
}
