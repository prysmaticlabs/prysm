package testing

import (
	"context"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// Operations defines a mock for the operations service.
type Operations struct {
	Attestations []*ethpb.Attestation
}

// AttestationPool --
func (op *Operations) AttestationPool(ctx context.Context, requestedSlot uint64) ([]*ethpb.Attestation, error) {
	return op.Attestations, nil
}

// AttestationPoolNoVerify --
func (op *Operations) AttestationPoolNoVerify(ctx context.Context) ([]*ethpb.Attestation, error) {
	return op.Attestations, nil
}

// AttestationPoolForForkchoice --
func (op *Operations) AttestationPoolForForkchoice(ctx context.Context) ([]*ethpb.Attestation, error) {
	return op.Attestations, nil
}

// HandleAttestation --
func (op *Operations) HandleAttestation(context.Context, proto.Message) error {
	return nil
}

// AttestationsBySlotCommittee --
func (op *Operations) AttestationsBySlotCommittee(ctx context.Context, slot uint64, index uint64) ([]*ethpb.Attestation, error) {
	return nil, nil
}
