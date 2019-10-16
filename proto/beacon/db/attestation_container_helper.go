package db

import (
	"github.com/prysmaticlabs/go-bitfield"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func NewContainerFromAttestations(atts []*ethpb.Attestation) *AttestationContainer {
	if len(atts) == 0 {
		panic("no attestations provided")
	}
	var sp []*AttestationContainer_SignaturePair
	for _, att := range atts {
		sp = append(sp, &AttestationContainer_SignaturePair{
			AggregationBits: att.AggregationBits,
			Signature:       att.Signature,
		})
	}
	return &AttestationContainer{
		Data:           atts[0].Data,
		SignaturePairs: sp,
	}
}

func (ac *AttestationContainer) Contains(att *ethpb.Attestation) bool {
	for _, sp := range ac.SignaturePairs {
		if sp.AggregationBits.Contains(att.AggregationBits) {
			return true
		}
	}
	return false
}

func ToAttestations(ac *AttestationContainer) []*ethpb.Attestation {
	if ac == nil {
		return nil
	}

	atts := make([]*ethpb.Attestation, len(ac.SignaturePairs))
	for i, sp := range ac.SignaturePairs {
		atts[i] = &ethpb.Attestation{
			Data:            ac.Data,
			AggregationBits: sp.AggregationBits,
			Signature:       sp.Signature,
			// TODO: Add custody bits in phase 1.
			// Stub: CustodyBits must be same length as aggregation bits; committee size.
			CustodyBits: bitfield.NewBitlist(sp.AggregationBits.Len()),
		}
	}
	return atts
}
