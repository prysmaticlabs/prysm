package client

import (
	"context"
	"io"
	"time"

	ptypes "github.com/gogo/protobuf/types"

	"github.com/opentracing/opentracing-go"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotticker"
)

// validator
//
// WIP - not done.
type validator struct {
	genesisTime     uint64
	ticker          slotticker.SlotTicker
	assignment      *pb.Assignment
	validatorClient pb.ValidatorServiceClient
	beaconClient    pb.BeaconServiceClient
	pubKey          []byte
}

// Initialize
//
// WIP - not done.
func (v *validator) Initialize(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.Initialize")
	defer span.Finish()
}

// Done cleans up the validator.
func (v *validator) Done() {
	v.ticker.Done()
}

// WaitForActivation checks whether the validator pubkey is in the active
// validator set. If not, this operation will block until an activation message is
// received.
//
// WIP - not done.
func (v *validator) WaitForActivation(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.WaitForActivation")
	defer span.Finish()
	// First, check if the beacon chain has started.
	stream, err := v.beaconClient.WaitForChainStart(ctx, &ptypes.Empty{})
	if err != nil {
		log.Errorf("Could not setup beacon chain ChainStart streaming client: %v", err)
		return
	}
	for {
		chainStartRes, err := stream.Recv()
		// If the stream is closed, we stop the loop.
		if err == io.EOF {
			break
		}
		// If context is canceled we stop the loop.
		if ctx.Err() == context.Canceled {
			log.Debugf("Context has been canceled so shutting down the loop: %v", ctx.Err())
			break
		}
		if err != nil {
			log.Errorf("Could not receive ChainStart from stream: %v", err)
			continue
		}
		v.genesisTime = chainStartRes.GenesisTime
	}
	// Once the ChainStart log is received, we update the genesis time of the validator client
	// and begin a slot ticker used to track the current slot the beacon node is in.
	v.ticker = slotticker.GetSlotTicker(time.Unix(int64(v.genesisTime), 0), params.BeaconConfig().SlotDuration)
	// Then, check if the validator has deposited into the Deposit Contract.
	// If the validator has deposited, subscribe to a stream receiving the activation status
	// of the validator until a final ACTIVATED check if received, then this function can return.
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
}

// AttestToBlockHead
//
// WIP - not done.
func (v *validator) AttestToBlockHead(ctx context.Context, slot uint64) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.AttestToBlockHead")
	defer span.Finish()
}
