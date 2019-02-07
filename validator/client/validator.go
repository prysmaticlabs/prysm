// Package client represents the functionality to act as a validator.
package client

import (
	"context"
	"io"
	"time"

	ptypes "github.com/gogo/protobuf/types"

	"github.com/opentracing/opentracing-go"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
)

// AttestationPool STUB interface. Final attestation pool pending design.
// TODO(1323): Replace with actual attestation pool.
type AttestationPool interface {
	PendingAttestations() []*pbp2p.Attestation
}

// validator
//
// WIP - not done.
type validator struct {
	genesisTime     uint64
	ticker          *slotutil.SlotTicker
	assignment      *pb.Assignment
	proposerClient  pb.ProposerServiceClient
	validatorClient pb.ValidatorServiceClient
	beaconClient    pb.BeaconServiceClient
	attesterClient  pb.AttesterServiceClient
	pubKey          []byte
	attestationPool AttestationPool
}

// Done cleans up the validator.
func (v *validator) Done() {
	v.ticker.Done()
}

// WaitForChainStart checks whether the beacon node has started its runtime. That is,
// it calls to the beacon node which then verifies the ETH1.0 deposit contract logs to check
// for the ChainStart log to have been emitted. If so, it starts a ticker based on the ChainStart
// unix timestamp which will be used to keep track of time within the validator client.
func (v *validator) WaitForChainStart(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.WaitForChainStart")
	defer span.Finish()
	// First, check if the beacon chain has started.
	stream, err := v.beaconClient.WaitForChainStart(ctx, &ptypes.Empty{})
	if err != nil {
		log.Errorf("Could not setup beacon chain ChainStart streaming client: %v", err)
		return
	}
	for {
		log.Info("Waiting for beacon chain start log from the ETH 1.0 deposit contract...")
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
		break
	}
	log.Infof("Beacon chain initialized at unix time: %v", time.Unix(int64(v.genesisTime), 0))
	// Once the ChainStart log is received, we update the genesis time of the validator client
	// and begin a slot ticker used to track the current slot the beacon node is in.
	v.ticker = slotutil.GetSlotTicker(time.Unix(int64(v.genesisTime), 0), params.BeaconConfig().SlotDuration)
}

// WaitForActivation checks whether the validator pubkey is in the active
// validator set. If not, this operation will block until an activation message is
// received.
//
// WIP - not done.
func (v *validator) WaitForActivation(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.WaitForActivation")
	defer span.Finish()
	// First, check if the validator has deposited into the Deposit Contract.
	// If the validator has deposited, subscribe to a stream receiving the activation status.
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
