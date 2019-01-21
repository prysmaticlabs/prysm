package client

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotticker"
)

// validator
//
// WIP - not done.
type validator struct {
	ticker      slotticker.SlotTicker
	assignments map[uint64]pb.ValidatorRole
}

// Initialize
//
// WIP - not done.
func (v *validator) Initialize() {
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
func (v *validator) WaitForActivation() {
}

// NextSlot emits the next slot number at the start time of that slot.
func (v *validator) NextSlot() <-chan uint64 {
	return v.ticker.C()
}

// UpdateAssignments checks the slot number to determine if the validator's
// list of upcoming assignments needs to be updated. For example, at the
// beginning of a new epoch.
func (v *validator) UpdateAssignments(slot uint64) {

}

// RoleAt slot returns the validator role at the given slot. Returns nil if the
// validator is known to not have a role at the at slot. Returns UNKNOWN if the
// validator assignments are unknown. Otherwise returns a valid ValidatorRole.
func (v *validator) RoleAt(slot uint64) pb.ValidatorRole {
	if v.assignments == nil {
		return pb.ValidatorRole_UNKNOWN
	}
	return v.assignments[slot]
}

// ProposeBlock
//
// WIP - not done.
func (v *validator) ProposeBlock(slot uint64) {

}

// AttestToBlockHead
//
// WIP - not done.
func (v *validator) AttestToBlockHead(slot uint64) {

}
