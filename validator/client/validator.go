// Package client represents the functionality to act as a validator.
package client

import (
	"context"
	"fmt"
	"io"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

type validator struct {
	genesisTime     uint64
	ticker          *slotutil.SlotTicker
	assignment      *pb.CommitteeAssignmentResponse
	proposerClient  pb.ProposerServiceClient
	validatorClient pb.ValidatorServiceClient
	beaconClient    pb.BeaconServiceClient
	attesterClient  pb.AttesterServiceClient
	key             *keystore.Key
	prevBalance     uint64
}

// Done cleans up the validator.
func (v *validator) Done() {
	v.ticker.Done()
}

// WaitForChainStart checks whether the beacon node has started its runtime. That is,
// it calls to the beacon node which then verifies the ETH1.0 deposit contract logs to check
// for the ChainStart log to have been emitted. If so, it starts a ticker based on the ChainStart
// unix timestamp which will be used to keep track of time within the validator client.
func (v *validator) WaitForChainStart(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "validator.WaitForChainStart")
	defer span.End()
	// First, check if the beacon chain has started.
	stream, err := v.beaconClient.WaitForChainStart(ctx, &ptypes.Empty{})
	if err != nil {
		return fmt.Errorf("could not setup beacon chain ChainStart streaming client: %v", err)
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
			return fmt.Errorf("context has been canceled so shutting down the loop: %v", ctx.Err())
		}
		if err != nil {
			return fmt.Errorf("could not receive ChainStart from stream: %v", err)
		}
		v.genesisTime = chainStartRes.GenesisTime
		break
	}
	// Once the ChainStart log is received, we update the genesis time of the validator client
	// and begin a slot ticker used to track the current slot the beacon node is in.
	v.ticker = slotutil.GetSlotTicker(time.Unix(int64(v.genesisTime), 0), params.BeaconConfig().SecondsPerSlot)
	log.Infof("Beacon chain initialized at unix time: %v", time.Unix(int64(v.genesisTime), 0))
	return nil
}

// WaitForActivation checks whether the validator pubkey is in the active
// validator set. If not, this operation will block until an activation message is
// received.
func (v *validator) WaitForActivation(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "validator.WaitForActivation")
	defer span.End()
	req := &pb.ValidatorActivationRequest{
		Pubkey: v.key.PublicKey.Marshal(),
	}
	stream, err := v.validatorClient.WaitForActivation(ctx, req)
	if err != nil {
		return fmt.Errorf("could not setup validator WaitForActivation streaming client: %v", err)
	}
	var validatorActivatedRecord *pbp2p.Validator
	for {
		log.Info("Waiting for validator to be activated in the beacon chain")
		res, err := stream.Recv()
		// If the stream is closed, we stop the loop.
		if err == io.EOF {
			break
		}
		// If context is canceled we stop the loop.
		if ctx.Err() == context.Canceled {
			return fmt.Errorf("context has been canceled so shutting down the loop: %v", ctx.Err())
		}
		if err != nil {
			return fmt.Errorf("could not receive validator activation from stream: %v", err)
		}
		validatorActivatedRecord = res.Validator
		break
	}
	log.WithFields(logrus.Fields{
		"activationEpoch": validatorActivatedRecord.ActivationEpoch - params.BeaconConfig().GenesisEpoch,
	}).Info("Validator activated")
	return nil
}

// CanonicalHeadSlot returns the slot of canonical block currently found in the
// beacon chain via RPC.
func (v *validator) CanonicalHeadSlot(ctx context.Context) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "validator.CanonicalHeadSlot")
	defer span.End()
	head, err := v.beaconClient.CanonicalHead(ctx, &ptypes.Empty{})
	if err != nil {
		return params.BeaconConfig().GenesisSlot, err
	}
	return head.Slot, nil
}

// NextSlot emits the next slot number at the start time of that slot.
func (v *validator) NextSlot() <-chan uint64 {
	return v.ticker.C()
}

// SlotDeadline is the start time of the next slot.
func (v *validator) SlotDeadline(slot uint64) time.Time {
	secs := (slot + 1 - params.BeaconConfig().GenesisSlot) * params.BeaconConfig().SecondsPerSlot
	return time.Unix(int64(v.genesisTime), 0 /*ns*/).Add(time.Duration(secs) * time.Second)
}

// UpdateAssignments checks the slot number to determine if the validator's
// list of upcoming assignments needs to be updated. For example, at the
// beginning of a new epoch.
func (v *validator) UpdateAssignments(ctx context.Context, slot uint64) error {
	// Testing run time for fetching every slot. This is not meant for production!
	// https://github.com/prysmaticlabs/prysm/issues/2167
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 && v.assignment != nil && false {
		// Do nothing if not epoch start AND assignments already exist.
		return nil
	}

	ctx, span := trace.StartSpan(ctx, "validator.UpdateAssignments")
	defer span.End()

	req := &pb.CommitteeAssignmentsRequest{
		EpochStart: slot,
		PublicKeys: [][]byte{v.key.PublicKey.Marshal()},
	}

	resp, err := v.validatorClient.CommitteeAssignment(ctx, req)
	if err != nil {
		v.assignment = nil // Clear assignments so we know to retry the request.
		return err
	}

	v.assignment = resp

	lFields := logrus.Fields{
		"attesterSlot": resp.Assignment[0].Slot - params.BeaconConfig().GenesisSlot,
		"proposerSlot": "Not proposing",
		"shard":        resp.Assignment[0].Shard,
	}
	if v.assignment.Assignment[0].IsProposer {
		lFields["proposerSlot"] = resp.Assignment[0].Slot - params.BeaconConfig().GenesisSlot
	}

	log.WithFields(lFields).Info("Updated validator assignments")
	return nil
}

// RoleAt slot returns the validator role at the given slot. Returns nil if the
// validator is known to not have a role at the at slot. Returns UNKNOWN if the
// validator assignments are unknown. Otherwise returns a valid ValidatorRole.
func (v *validator) RoleAt(slot uint64) pb.ValidatorRole {
	if v.assignment == nil {
		return pb.ValidatorRole_UNKNOWN
	}
	if v.assignment.Assignment[0].Slot == slot {
		if v.assignment.Assignment[0].IsProposer {
			// Note: A proposer also attests to the slot.
			return pb.ValidatorRole_PROPOSER
		}
		return pb.ValidatorRole_ATTESTER
	}
	return pb.ValidatorRole_UNKNOWN
}
