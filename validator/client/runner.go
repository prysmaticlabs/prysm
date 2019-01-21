package client

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
)

// Validator interface defines the primary methods of a validator client.
type Validator interface {
	Initialize()
	Done()
	WaitForActivation()
	NextSlot() <-chan uint64
	UpdateAssignments(slot uint64)
	RoleAt(slot uint64) pb.ValidatorRole
	AttestToBlockHead(slot uint64)
	ProposeBlock(slot uint64)
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
	v.Initialize()
	defer v.Done()
	v.WaitForActivation()

	for {
		select {
		case <-ctx.Done():
			log.Info("Context cancelled, stopping validator")
			return // Exit if context is cancelled.
		case slot := <-v.NextSlot():
			v.UpdateAssignments(slot)
			role := v.RoleAt(slot)

			switch role {
			case pb.ValidatorRole_ATTESTER:
				v.AttestToBlockHead(slot)
				break
			case pb.ValidatorRole_PROPOSER:
				v.ProposeBlock(slot)
				break
			case pb.ValidatorRole_UNKNOWN:
				// This shouldn't happen normally, so it is considered a warning.
				log.WithFields(logrus.Fields{
					"slot": slot,
					"role": role,
				}).Warn("Unknown role, doing nothing")
			default:
				// Do nothing :)
				break
			}
		}
	}
}
