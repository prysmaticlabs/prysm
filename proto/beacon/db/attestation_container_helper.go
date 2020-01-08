package db

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
)

// Contains returns true if the attestation bits are fully contained in some attestations.
func (ac *AttestationContainer) Contains(att *ethpb.Attestation) bool {
	all := bitfield.NewBitlist(att.AggregationBits.Len())
	for _, sp := range ac.SignaturePairs {
		if all.Len() != sp.AggregationBits.Len() {
			continue
		}
		all = all.Or(sp.AggregationBits)
	}
	return all.Contains(att.AggregationBits)
}

// ToAttestations converts an attestationContainer signature pairs to full attestations.
func (ac *AttestationContainer) ToAttestations() []*ethpb.Attestation {
	if ac == nil {
		return nil
	}

	atts := make([]*ethpb.Attestation, len(ac.SignaturePairs))
	for i, sp := range ac.SignaturePairs {
		atts[i] = &ethpb.Attestation{
			Data:            ac.Data,
			AggregationBits: sp.AggregationBits,
			Signature:       sp.Signature,
		}
	}
	return atts
}

// InsertAttestation if bitfields do not exist already.
func (ac *AttestationContainer) InsertAttestation(att *ethpb.Attestation) {
	sigPairsNotEclipsed := make([]*AttestationContainer_SignaturePair, 0, len(ac.SignaturePairs))
	// if att is fully contained in some existing bitfields, do nothing.
	if ac.Contains(att) {
		return
	}

	for _, sp := range ac.SignaturePairs {
		// filter any existing signature pairs that are fully contained within
		// the new attestation.
		if !att.AggregationBits.Contains(sp.AggregationBits) {
			sigPairsNotEclipsed = append(sigPairsNotEclipsed, sp)
		}
	}
	ac.SignaturePairs = append(sigPairsNotEclipsed, &AttestationContainer_SignaturePair{
		AggregationBits: att.AggregationBits,
		Signature:       att.Signature,
	})
}
