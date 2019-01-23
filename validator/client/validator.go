package client

import (
	"context"

	"github.com/opentracing/opentracing-go"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotticker"
)

// validator
//
// WIP - not done.
type validator struct {
	ticker      slotticker.SlotTicker
	assignments map[uint64]*pb.Assignment

	validatorClient pb.ValidatorServiceClient
	pubKey          *pb.PublicKey
}

// Initialize
//
// WIP - not done.
func (v *validator) Initialize(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.Initialize")
	defer span.Finish()

	cfg := params.BeaconConfig()
	v.ticker = slotticker.GetSlotTicker(cfg.GenesisTime, cfg.SlotDuration)
}

// Done cleans up the validator.
func (v *validator) Done() {
	v.ticker.Done()
}

// WaitForActivation checks whether the validator pubkey is in the active
// validator set. If not, this operation will block until an activation is
// received.
//
// WIP - not done.
func (v *validator) WaitForActivation(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.WaitForActivation")
	defer span.Finish()
}

// NextSlot emits the next slot number at the start time of that slot.
func (v *validator) NextSlot() <-chan uint64 {
	return v.ticker.C()
}

// UpdateAssignments checks the slot number to determine if the validator's
// list of upcoming assignments needs to be updated. For example, at the
// beginning of a new epoch.
func (v *validator) UpdateAssignments(ctx context.Context, slot uint64) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.UpdateAssignments")
	defer span.Finish()

	req := &pb.ValidatorEpochAssignmentsRequest{
		EpochStart: slot,
		PublicKey:  v.pubKey,
	}

	resp, err := v.validatorClient.ValidatorEpochAssignments(ctx, req)
	if err != nil {
		return err
	}

	m := make(map[uint64]*pb.Assignment)
	for _, a := range resp.Assignments {
		m[a.AssignedSlot] = a
	}
	v.assignments = m

	return nil
}

// RoleAt slot returns the validator role at the given slot. Returns nil if the
// validator is known to not have a role at the at slot. Returns UNKNOWN if the
// validator assignments are unknown. Otherwise returns a valid ValidatorRole.
func (v *validator) RoleAt(slot uint64) pb.ValidatorRole {
	if v.assignments == nil {
		return pb.ValidatorRole_UNKNOWN
	}
	if v.assignments[slot] == nil {
		return pb.ValidatorRole_UNKNOWN
	}
	return v.assignments[slot].Role
}

// ProposeBlock
//
// WIP - not done.
func (v *validator) ProposeBlock(ctx context.Context, slot uint64) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.ProposeBlock")
	defer span.Finish()

}

// AttestToBlockHead
//
// WIP - not done.
func (v *validator) AttestToBlockHead(ctx context.Context, slot uint64) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.AttestToBlockHead")
	defer span.Finish()

}
