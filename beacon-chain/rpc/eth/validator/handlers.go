package validator

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/network"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// GetAggregateAttestation aggregates all attestations matching the given attestation data root and slot, returning the aggregated result.
func (s *Server) GetAggregateAttestation(w http.ResponseWriter, r *http.Request) {
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

	if err := s.AttestationsPool.AggregateUnaggregatedAttestations(r.Context()); err != nil {
		errJson := &network.DefaultErrorJson{
			Message: "Could not aggregate unaggregated attestations: " + err.Error(),
			Code:    http.StatusBadRequest,
		}
		network.WriteError(w, errJson)
		return
	}

	allAtts := s.AttestationsPool.AggregatedAttestations()
	var bestMatchingAtt *ethpbalpha.Attestation
	for _, att := range allAtts {
		if att.Data.Slot == primitives.Slot(slot) {
			root, err := att.Data.HashTreeRoot()
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: "Could not get attestation data root: " + err.Error(),
					Code:    http.StatusInternalServerError,
				}
				network.WriteError(w, errJson)
				return
			}
			attDataRootBytes, err := hexutil.Decode(attDataRoot)
			if err != nil {
				errJson := &network.DefaultErrorJson{
					Message: "Could not decode attestation data root into bytes: " + err.Error(),
					Code:    http.StatusBadRequest,
				}
				network.WriteError(w, errJson)
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
		errJson := &network.DefaultErrorJson{
			Message: "No matching attestation found",
			Code:    http.StatusNotFound,
		}
		network.WriteError(w, errJson)
		return
	}

	response := &AggregateAttestationResponse{
		Data: shared.Attestation{
			AggregationBits: hexutil.Encode(bestMatchingAtt.AggregationBits),
			Data: shared.AttestationData{
				Slot:            strconv.FormatUint(uint64(bestMatchingAtt.Data.Slot), 10),
				CommitteeIndex:  strconv.FormatUint(uint64(bestMatchingAtt.Data.CommitteeIndex), 10),
				BeaconBlockRoot: hexutil.Encode(bestMatchingAtt.Data.BeaconBlockRoot),
				Source: shared.Checkpoint{
					Epoch: strconv.FormatUint(uint64(bestMatchingAtt.Data.Source.Epoch), 10),
					Root:  hexutil.Encode(bestMatchingAtt.Data.Source.Root),
				},
				Target: shared.Checkpoint{
					Epoch: strconv.FormatUint(uint64(bestMatchingAtt.Data.Target.Epoch), 10),
					Root:  hexutil.Encode(bestMatchingAtt.Data.Target.Root),
				},
			},
			Signature: hexutil.Encode(bestMatchingAtt.Signature),
		}}
	network.WriteJson(w, response)
}
