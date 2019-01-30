package client

import (
	"context"

	"github.com/opentracing/opentracing-go"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotticker"
)

type AttestationPool interface {
	PendingAttestations() []*pbp2p.Attestation
}

type BlockThing interface {
	HeadBlock() *pbp2p.BeaconBlock
}

// validator
//
// WIP - not done.
type validator struct {
	ticker     slotticker.SlotTicker
	assignment *pb.Assignment

	validatorClient pb.ValidatorServiceClient
	pubKey          []byte
	attestationPool AttestationPool
	p2p             p2p.Broadcaster
	blockThing      BlockThing
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

	if slot%params.BeaconConfig().EpochLength != 0 {
		// Do nothing if not epoch start.
		return nil
	}

	req := &pb.ValidatorEpochAssignmentsRequest{
		EpochStart: slot,
		PublicKey:  v.pubKey,
	}

	resp, err := v.validatorClient.ValidatorEpochAssignments(ctx, req)
	if err != nil {
		return err
	}

	v.assignment = resp.Assignment
	return nil
}

// RoleAt slot returns the validator role at the given slot. Returns nil if the
// validator is known to not have a role at the at slot. Returns UNKNOWN if the
// validator assignments are unknown. Otherwise returns a valid ValidatorRole.
func (v *validator) RoleAt(slot uint64) pb.ValidatorRole {
	if v.assignment == nil {
		return pb.ValidatorRole_UNKNOWN
	}
	if v.assignment.AttesterSlot == slot {
		return pb.ValidatorRole_ATTESTER
	} else if v.assignment.ProposerSlot == slot {
		return pb.ValidatorRole_PROPOSER
	}
	return pb.ValidatorRole_UNKNOWN
}

// ProposeBlock
//
// WIP - not done.
func (v *validator) ProposeBlock(ctx context.Context, slot uint64) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.ProposeBlock")
	defer span.Finish()

	// 1. Get current head beacon block.
	headBlock := v.blockThing.HeadBlock()
	parentHash, err := hashutil.HashBeaconBlock(headBlock)
	if err != nil {
		log.Errorf("Failed to hash parent block: %v", err)
		return
	}

	// 2. Construct block
	block := &pbp2p.BeaconBlock{
		Slot:               slot,
		ParentRootHash32:   parentHash[:], // tree root? in ssz pkg
		RandaoRevealHash32: nil,           // TODO: generate randao reveal
		Eth1Data: &pbp2p.Eth1Data{ // TODO(raul): will write rpc
			DepositRootHash32: nil,
			BlockHash32:       nil,
		},
		Body: &pbp2p.BeaconBlockBody{
			Attestations: v.attestationPool.PendingAttestations(),
			// TODO: slashings
			ProposerSlashings: nil, // TODO later
			CasperSlashings:   nil, // TODO later
			Deposits:          nil, // TODO(raul): fetch from gRPC
			Exits:             nil, // TODO: later
		},
	}

	block.StateRootHash32 = nil // TODO(raul): Run state transition on unsigned block

	// TODO: sign block
	block.Signature = nil

	v.p2p.Broadcast(block)
}

// AttestToBlockHead
//
// WIP - not done.
func (v *validator) AttestToBlockHead(ctx context.Context, slot uint64) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.AttestToBlockHead")
	defer span.Finish()

}
