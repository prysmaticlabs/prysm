package client

import (
	"context"
	"sync"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validator interface defines the primary methods of a validator client.
type Validator interface {
	Done()
	WaitForChainStart(ctx context.Context) error
	WaitForActivation(ctx context.Context) error
	WaitForSync(ctx context.Context) error
	CanonicalHeadSlot(ctx context.Context) (uint64, error)
	NextSlot() <-chan uint64
	SlotDeadline(slot uint64) time.Time
	LogValidatorGainsAndLosses(ctx context.Context, slot uint64) error
	UpdateDuties(ctx context.Context, slot uint64) error
	RolesAt(ctx context.Context, slot uint64) (map[[48]byte][]pb.ValidatorRole, error) // validator pubKey -> roles
	SubmitAttestation(ctx context.Context, slot uint64, pubKey [48]byte)
	ProposeBlock(ctx context.Context, slot uint64, pubKey [48]byte)
	SubmitAggregateAndProof(ctx context.Context, slot uint64, pubKey [48]byte)
	LogAttestationsSubmitted()
	UpdateDomainDataCaches(ctx context.Context, slot uint64)
}

// Run the main validator routine. This routine exits if the context is
// canceled.
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
	if err := v.WaitForSync(ctx); err != nil {
		log.Fatalf("Could not determine if beacon node synced: %v", err)
	}
	if err := v.WaitForActivation(ctx); err != nil {
		log.Fatalf("Could not wait for validator activation: %v", err)
	}
	headSlot, err := v.CanonicalHeadSlot(ctx)
	if err != nil {
		log.Fatalf("Could not get current canonical head slot: %v", err)
	}
	if err := v.UpdateDuties(ctx, headSlot); err != nil {
		handleAssignmentError(err, headSlot)
	}
	for {
		ctx, span := trace.StartSpan(ctx, "validator.processSlot")

		select {
		case <-ctx.Done():
			log.Info("Context canceled, stopping validator")
			return // Exit if context is canceled.
		case slot := <-v.NextSlot():
			span.AddAttributes(trace.Int64Attribute("slot", int64(slot)))
			slotCtx, cancel := context.WithDeadline(ctx, v.SlotDeadline(slot))
			// Report this validator client's rewards and penalties throughout its lifecycle.
			log := log.WithField("slot", slot)
			if err := v.LogValidatorGainsAndLosses(slotCtx, slot); err != nil {
				log.WithError(err).Error("Could not report validator's rewards/penalties")
			}

			// Keep trying to update assignments if they are nil or if we are past an
			// epoch transition in the beacon node's state.
			if err := v.UpdateDuties(ctx, slot); err != nil {
				handleAssignmentError(err, slot)
				cancel()
				span.End()
				continue
			}

			// Start fetching domain data for the next epoch.
			if helpers.IsEpochEnd(slot) {
				go v.UpdateDomainDataCaches(ctx, slot+1)
			}

			var wg sync.WaitGroup

			allRoles, err := v.RolesAt(ctx, slot)
			if err != nil {
				log.WithError(err).Error("Could not get validator roles")
				continue
			}
			for id, roles := range allRoles {
				wg.Add(1)
				go func(roles []pb.ValidatorRole, id [48]byte) {
					for _, role := range roles {
						switch role {
						case pb.ValidatorRole_ATTESTER:
							go v.SubmitAttestation(slotCtx, slot, id)
						case pb.ValidatorRole_PROPOSER:
							go v.ProposeBlock(slotCtx, slot, id)
						case pb.ValidatorRole_AGGREGATOR:
							go v.SubmitAggregateAndProof(slotCtx, slot, id)
						case pb.ValidatorRole_UNKNOWN:
							log.Debug("No active roles, doing nothing")
						default:
							log.Warnf("Unhandled role %v", role)
						}
					}
					wg.Done()
				}(roles, id)
			}
			// Wait for all processes to complete, then report span complete.
			go func() {
				wg.Wait()
				v.LogAttestationsSubmitted()
				span.End()
			}()
		}
	}
}

func handleAssignmentError(err error, slot uint64) {
	if errCode, ok := status.FromError(err); ok && errCode.Code() == codes.NotFound {
		log.WithField(
			"epoch", slot/params.BeaconConfig().SlotsPerEpoch,
		).Warn("Validator not yet assigned to epoch")
	} else {
		log.WithField("error", err).Error("Failed to update assignments")
	}
}
