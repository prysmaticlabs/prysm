package helpers

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

var (
	// ErrAttestationDataSlotNilState is returned when a nil state argument
	// is provided to AttestationDataSlot.
	ErrAttestationDataSlotNilState = errors.New("nil state provided for AttestationDataSlot")
	// ErrAttestationDataSlotNilData is returned when a nil attestation data
	// argument is provided to AttestationDataSlot.
	ErrAttestationDataSlotNilData = errors.New("nil data provided for AttestationDataSlot")
)

// AggregateAttestation aggregates attestations a1 and a2 together.
func AggregateAttestation(a1 *ethpb.Attestation, a2 *ethpb.Attestation) (*ethpb.Attestation, error) {
	baseAtt := proto.Clone(a1).(*ethpb.Attestation)
	newAtt := proto.Clone(a2).(*ethpb.Attestation)
	if newAtt.AggregationBits.Count() > baseAtt.AggregationBits.Count() {
		baseAtt, newAtt = newAtt, baseAtt
	}

	if baseAtt.AggregationBits.Contains(newAtt.AggregationBits) {
		return baseAtt, nil
	}

	newBits := baseAtt.AggregationBits.Or(newAtt.AggregationBits)
	newSig, err := bls.SignatureFromBytes(newAtt.Signature)
	if err != nil {
		return nil, err
	}
	baseSig, err := bls.SignatureFromBytes(baseAtt.Signature)
	if err != nil {
		return nil, err
	}

	aggregatedSig := bls.AggregateSignatures([]*bls.Signature{baseSig, newSig})
	baseAtt.Signature = aggregatedSig.Marshal()
	baseAtt.AggregationBits = newBits

	return baseAtt, nil
}
