package client

import (
	"context"
	"github.com/opentracing/opentracing-go"
)

// AttestToBlockHead
//
// WIP - not done.
func (v *validator) AttestToBlockHead(ctx context.Context, slot uint64) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.AttestToBlockHead")
	defer span.Finish()
	// First the validator should construct attestation_data, an AttestationData
	// object based upon the state at the assigned slot.
}
