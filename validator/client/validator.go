// Package client represents the functionality to act as a validator.
package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

type validator struct {
	genesisTime          uint64
	ticker               *slotutil.SlotTicker
	assignments          *pb.CommitteeAssignmentResponse
	proposerClient       pb.ProposerServiceClient
	validatorClient      pb.ValidatorServiceClient
	beaconClient         pb.BeaconServiceClient
	attesterClient       pb.AttesterServiceClient
	keys                 map[string]*keystore.Key
	pubkeys              [][]byte
	prevBalance          uint64
	logValidatorBalances bool
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
	log.WithField("genesisTime", time.Unix(int64(v.genesisTime), 0)).Info("Beacon chain initialized")
	return nil
}

// WaitForActivation checks whether the validator pubkey is in the active
// validator set. If not, this operation will block until an activation message is
// received.
func (v *validator) WaitForActivation(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "validator.WaitForActivation")
	defer span.End()
	req := &pb.ValidatorActivationRequest{
		PublicKeys: v.pubkeys,
	}
	stream, err := v.validatorClient.WaitForActivation(ctx, req)
	if err != nil {
		return fmt.Errorf("could not setup validator WaitForActivation streaming client: %v", err)
	}
	var validatorActivatedRecords [][]byte
	for {
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
		log.Info("Waiting for validator to be activated in the beacon chain")
		activatedKeys := v.checkAndLogValidatorStatus(res.Statuses)

		if len(activatedKeys) > 0 {
			validatorActivatedRecords = activatedKeys
			break
		}
	}
	for _, pk := range validatorActivatedRecords {
		log.WithFields(logrus.Fields{
			"publicKey": fmt.Sprintf("%#x", pk),
		}).Info("Validator activated")
	}
	v.ticker = slotutil.GetSlotTicker(time.Unix(int64(v.genesisTime), 0), params.BeaconConfig().SecondsPerSlot)

	return nil
}

func (v *validator) checkAndLogValidatorStatus(validatorStatuses []*pb.ValidatorActivationResponse_Status) [][]byte {
	var activatedKeys [][]byte
	for _, status := range validatorStatuses {
		if status.Status.Status == pb.ValidatorStatus_ACTIVE {
			activatedKeys = append(activatedKeys, status.PublicKey)
		}
		if status.Status.DepositInclusionSlot == 0 {
			log.WithFields(logrus.Fields{
				"publicKey": fmt.Sprintf("%#x", bytesutil.Trunc(status.PublicKey)),
				"status":    fmt.Sprintf("%s", status.Status.Status.String()),
			}).Info("Not yet included in state...")
			continue
		}
		if status.Status.ActivationEpoch == (params.BeaconConfig().FarFutureEpoch - params.BeaconConfig().GenesisEpoch) {
			log.WithFields(logrus.Fields{
				"publicKey":                 fmt.Sprintf("%#x", bytesutil.Trunc(status.PublicKey)),
				"status":                    status.Status.Status.String(),
				"depositInclusionSlot":      status.Status.DepositInclusionSlot,
				"positionInActivationQueue": status.Status.PositionInActivationQueue,
			}).Info("Waiting to be activated")
			continue
		}
		log.WithFields(logrus.Fields{
			"publicKey":                 fmt.Sprintf("%#x", bytesutil.Trunc(status.PublicKey)),
			"status":                    status.Status.Status.String(),
			"depositInclusionSlot":      status.Status.DepositInclusionSlot,
			"activationEpoch":           status.Status.ActivationEpoch,
			"positionInActivationQueue": status.Status.PositionInActivationQueue,
		}).Info("Validator status")
	}
	return activatedKeys
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
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 && v.assignments != nil {
		// Do nothing if not epoch start AND assignments already exist.
		return nil
	}

	ctx, span := trace.StartSpan(ctx, "validator.UpdateAssignments")
	defer span.End()

	req := &pb.CommitteeAssignmentsRequest{
		EpochStart: slot,
		PublicKeys: v.pubkeys,
	}

	resp, err := v.validatorClient.CommitteeAssignment(ctx, req)
	if err != nil {
		v.assignments = nil // Clear assignments so we know to retry the request.
		return err
	}

	v.assignments = resp
	// Only log the full assignments output on epoch start to be less verbose.
	if slot%params.BeaconConfig().SlotsPerEpoch == 0 {
		for _, assignment := range v.assignments.Assignment {
			var proposerSlot uint64
			var attesterSlot uint64
			assignmentKey := hex.EncodeToString(assignment.PublicKey)
			assignmentKey = assignmentKey[:12]
			lFields := logrus.Fields{
				"validator": assignmentKey,
				"status":    assignment.Status,
			}
			if assignment.Status != pb.ValidatorStatus_ACTIVE {
				log.WithFields(lFields).Info("New assignment")
				continue
			} else if assignment.IsProposer {
				proposerSlot = assignment.Slot
				attesterSlot = assignment.Slot
			} else {
				attesterSlot = assignment.Slot
			}
			lFields["attesterSlot"] = attesterSlot - params.BeaconConfig().GenesisSlot
			lFields["proposerSlot"] = "Not proposing"
			lFields["shard"] = assignment.Shard

			if assignment.IsProposer {
				lFields["proposerSlot"] = proposerSlot - params.BeaconConfig().GenesisSlot
			}
			log.WithFields(lFields).Info("New assignment")

		}
	}

	log.WithFields(logrus.Fields{
		"assignments": len(v.assignments.Assignment),
	}).Info("Updated validator assignments")

	return nil
}

// RolesAt slot returns the validator roles at the given slot. Returns nil if the
// validator is known to not have a roles at the at slot. Returns UNKNOWN if the
// validator assignments are unknown. Otherwise returns a valid ValidatorRole map.
func (v *validator) RolesAt(slot uint64) map[string]pb.ValidatorRole {
	rolesAt := make(map[string]pb.ValidatorRole)
	for _, assignment := range v.assignments.Assignment {
		var role pb.ValidatorRole
		if assignment == nil {
			role = pb.ValidatorRole_UNKNOWN
		}
		if assignment.Slot == slot {
			// Note: A proposer also attests to the slot.
			if assignment.IsProposer {
				role = pb.ValidatorRole_PROPOSER
			} else {
				role = pb.ValidatorRole_ATTESTER
			}
		} else {
			role = pb.ValidatorRole_UNKNOWN
		}
		rolesAt[hex.EncodeToString(assignment.PublicKey)] = role
	}
	return rolesAt
}
