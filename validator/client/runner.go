package client

import (
	"context"
	"time"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validator interface defines the primary methods of a validator client.
type Validator interface {
	Done()
	WaitForChainStart(ctx context.Context) error
	WaitForActivation(ctx context.Context) error
	CanonicalHeadSlot(ctx context.Context) (uint64, error)
	NextSlot() <-chan uint64
	SlotDeadline(slot uint64) time.Time
	LogValidatorGainsAndLosses(ctx context.Context, slot uint64) error
	UpdateAssignments(ctx context.Context, slot uint64) error
	RoleAt(slot uint64) pb.ValidatorRole
	AttestToBlockHead(ctx context.Context, slot uint64)
	ProposeBlock(ctx context.Context, slot uint64)
}

// Run the main validator routine. This routine exits if the context is
// cancelled.
//
// Order of operations:
// 1 - Initialize validator data
// 2 - Wait for validator activation
// 3 - Wait for the next slot start
// 4 - Update assignments
// 5 - Determine role at current slot
// 6 - Perform assigned role, if any
func run(ctx context.Context, v Validator) {
	defer v.Done()
	if err := v.WaitForChainStart(ctx); err != nil {
		log.Fatalf("Could not determine if beacon chain started: %v", err)
	}
	if err := v.WaitForActivation(ctx); err != nil {
		log.Fatalf("Could not wait for validator activation: %v", err)
	}
	headSlot, err := v.CanonicalHeadSlot(ctx)
	if err != nil {
		log.Fatalf("Could not get current canonical head slot: %v", err)
	}
	if err := v.UpdateAssignments(ctx, headSlot); err != nil {
		handleAssignmentError(err, headSlot)
	}
	for {
		ctx, span := trace.StartSpan(ctx, "processSlot")
		defer span.End()

		select {
		case <-ctx.Done():
			log.Info("Context canceled, stopping validator")
			return // Exit if context is canceled.
		case slot := <-v.NextSlot():
			span.AddAttributes(trace.Int64Attribute("slot", int64(slot)))
			slotCtx, _ := context.WithDeadline(ctx, v.SlotDeadline(slot))
			// Report this validator client's rewards and penalties throughout its lifecycle.
			if err := v.LogValidatorGainsAndLosses(slotCtx, slot); err != nil {
				log.Errorf("Could not report validator's rewards/penalties for slot %d: %v",
					slot-params.BeaconConfig().GenesisSlot, err)
			}

			// Keep trying to update assignments if they are nil or if we are past an
			// epoch transition in the beacon node's state.
			if err := v.UpdateAssignments(slotCtx, slot); err != nil {
				handleAssignmentError(err, slot)
				continue
			}
			role := v.RoleAt(slot)

			switch role {
			case pb.ValidatorRole_ATTESTER:
				v.AttestToBlockHead(slotCtx, slot)
			case pb.ValidatorRole_PROPOSER:
				v.ProposeBlock(slotCtx, slot)
				v.AttestToBlockHead(slotCtx, slot)
			case pb.ValidatorRole_UNKNOWN:
				log.WithFields(logrus.Fields{
					"slot": slot - params.BeaconConfig().GenesisSlot,
					"role": role,
				}).Info("No active assignment, doing nothing")
			default:
				// Do nothing :)
			}
		}
	}
}

func handleAssignmentError(err error, slot uint64) {
	if errCode, ok := status.FromError(err); ok && errCode.Code() == codes.NotFound {
		log.Warnf("Validator not yet assigned between slots %d - %d",
			slot-params.BeaconConfig().GenesisSlot,
			slot+params.BeaconConfig().SlotsPerEpoch-params.BeaconConfig().GenesisSlot)
	} else {
		log.WithField("error", err).Error("Failed to update assignments")
	}
}
