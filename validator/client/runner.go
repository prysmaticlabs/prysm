package client

import (
	"context"

	"github.com/opentracing/opentracing-go"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
)

// Validator interface defines the primary methods of a validator client.
type Validator interface {
	Done()
	WaitForChainStart(ctx context.Context)
	WaitForActivation(ctx context.Context)
	NextSlot() <-chan uint64
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
	v.WaitForChainStart(ctx)
	v.WaitForActivation(ctx)
	span, ctx := opentracing.StartSpanFromContext(ctx, "processSlot")
	defer span.Finish()
	for {
		select {
		case <-ctx.Done():
			log.Info("Context cancelled, stopping validator")
			return // Exit if context is cancelled.
		case slot := <-v.NextSlot():
			if err := v.UpdateAssignments(ctx, slot); err != nil {
				log.WithField("error", err).Error("Failed to update assignments")
				continue
			}
			role := v.RoleAt(slot)

			switch role {
			case pb.ValidatorRole_ATTESTER:
				v.AttestToBlockHead(ctx, slot)
			case pb.ValidatorRole_PROPOSER:
				v.ProposeBlock(ctx, slot)
			case pb.ValidatorRole_UNKNOWN:
				// This shouldn't happen normally, so it is considered a warning.
				log.WithFields(logrus.Fields{
					"slot": slot,
					"role": role,
				}).Warn("Unknown role, doing nothing")
			default:
				// Do nothing :)
			}
		}
	}
}
