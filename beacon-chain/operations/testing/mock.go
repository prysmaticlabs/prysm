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

// HandleAttestation --
func (op *Operations) HandleAttestation(context.Context, proto.Message) error {
	return nil
}
