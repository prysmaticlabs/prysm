// Package client represents the functionality to act as a validator.
package client

import (
	"context"
	"fmt"
	"io"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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
	assignments          *pb.AssignmentResponse
	proposerClient       pb.ProposerServiceClient
	validatorClient      pb.ValidatorServiceClient
	attesterClient       pb.AttesterServiceClient
	node                 ethpb.NodeClient
	keys                 map[[48]byte]*keystore.Key
	pubkeys              [][]byte
	prevBalance          map[[48]byte]uint64
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
	stream, err := v.validatorClient.WaitForChainStart(ctx, &ptypes.Empty{})
	if err != nil {
		return errors.Wrap(err, "could not setup beacon chain ChainStart streaming client")
	}
	for {
		log.Info("Waiting for beacon chain start log from the ETH 1.0 deposit contract")
		chainStartRes, err := stream.Recv()
		// If the stream is closed, we stop the loop.
		if err == io.EOF {
			break
		}
		// If context is canceled we stop the loop.
		if ctx.Err() == context.Canceled {
			return errors.Wrap(ctx.Err(), "context has been canceled so shutting down the loop")
		}
		if err != nil {
			return errors.Wrap(err, "could not receive ChainStart from stream")
		}
		v.genesisTime = chainStartRes.GenesisTime
		break
	}
	// Once the ChainStart log is received, we update the genesis time of the validator client
	// and begin a slot ticker used to track the current slot the beacon node is in.
	v.ticker = slotutil.GetSlotTicker(time.Unix(int64(v.genesisTime), 0), params.BeaconConfig().SecondsPerSlot)
	log.WithField("genesisTime", time.Unix(int64(v.genesisTime), 0)).Info("Beacon chain started")
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
		return errors.Wrap(err, "could not setup validator WaitForActivation streaming client")
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
			return errors.Wrap(ctx.Err(), "context has been canceled so shutting down the loop")
		}
		if err != nil {
			return errors.Wrap(err, "could not receive validator activation from stream")
		}
		log.Info("Waiting for validator to be activated in the beacon chain")
		activatedKeys := v.checkAndLogValidatorStatus(res.Statuses)

		if len(activatedKeys) > 0 {
			validatorActivatedRecords = activatedKeys
			break
		}
	}
	for _, pubKey := range validatorActivatedRecords {
		log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:]))).Info("Validator activated")
	}
	v.ticker = slotutil.GetSlotTicker(time.Unix(int64(v.genesisTime), 0), params.BeaconConfig().SecondsPerSlot)

	return nil
}

// WaitForSync checks whether the beacon node has sync to the latest head
func (v *validator) WaitForSync(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "validator.WaitForSync")
	defer span.End()

	s, err := v.node.GetSyncStatus(ctx, &ptypes.Empty{})
	if err != nil {
		return errors.Wrap(err, "could not get sync status")
	}
	if !s.Syncing {
		return nil
	}

	for {
		select {
		// Poll every half slot
		case <-time.After(time.Duration(params.BeaconConfig().SlotsPerEpoch/2) * time.Second):
			s, err := v.node.GetSyncStatus(ctx, &ptypes.Empty{})
			if err != nil {
				return errors.Wrap(err, "could not get sync status")
			}
			if !s.Syncing {
				return nil
			}
			log.Info("Waiting for beacon node to sync to latest chain head")
		case <-ctx.Done():
			return errors.New("context has been canceled, exiting goroutine")
		}
	}
}

func (v *validator) checkAndLogValidatorStatus(validatorStatuses []*pb.ValidatorActivationResponse_Status) [][]byte {
	var activatedKeys [][]byte
	for _, status := range validatorStatuses {
		log := log.WithFields(logrus.Fields{
			"pubKey": fmt.Sprintf("%#x", bytesutil.Trunc(status.PublicKey[:])),
			"status": status.Status.Status.String(),
		})
		if status.Status.Status == pb.ValidatorStatus_ACTIVE {
			activatedKeys = append(activatedKeys, status.PublicKey)
			continue
		}
		if status.Status.Status == pb.ValidatorStatus_EXITED {
			log.Info("Validator exited")
			continue
		}
		if status.Status.Status == pb.ValidatorStatus_DEPOSIT_RECEIVED {
			log.WithField("expectedInclusionSlot", status.Status.DepositInclusionSlot).Info(
				"Deposit for validator received but not processed into state")
			continue
		}
		if status.Status.ActivationEpoch == params.BeaconConfig().FarFutureEpoch {
			log.WithFields(logrus.Fields{
				"depositInclusionSlot":      status.Status.DepositInclusionSlot,
				"positionInActivationQueue": status.Status.PositionInActivationQueue,
			}).Info("Waiting to be activated")
			continue
		}
		log.WithFields(logrus.Fields{
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
	head, err := v.validatorClient.CanonicalHead(ctx, &ptypes.Empty{})
	if err != nil {
		return 0, err
	}
	return head.Slot, nil
}

// NextSlot emits the next slot number at the start time of that slot.
func (v *validator) NextSlot() <-chan uint64 {
	return v.ticker.C()
}

// SlotDeadline is the start time of the next slot.
func (v *validator) SlotDeadline(slot uint64) time.Time {
	secs := (slot + 1) * params.BeaconConfig().SecondsPerSlot
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
	// Set deadline to end of epoch.
	ctx, cancel := context.WithDeadline(ctx, v.SlotDeadline(helpers.StartSlot(helpers.SlotToEpoch(slot)+1)))
	defer cancel()
	ctx, span := trace.StartSpan(ctx, "validator.UpdateAssignments")
	defer span.End()

	req := &pb.AssignmentRequest{
		EpochStart: slot / params.BeaconConfig().SlotsPerEpoch,
		PublicKeys: v.pubkeys,
	}

	resp, err := v.validatorClient.CommitteeAssignment(ctx, req)
	if err != nil {
		v.assignments = nil // Clear assignments so we know to retry the request.
		log.Error(err)
		return err
	}

	v.assignments = resp
	// Only log the full assignments output on epoch start to be less verbose.
	if slot%params.BeaconConfig().SlotsPerEpoch == 0 {
		for _, assignment := range v.assignments.ValidatorAssignment {
			lFields := logrus.Fields{
				"pubKey": fmt.Sprintf("%#x", bytesutil.Trunc(assignment.PublicKey)),
				"epoch":  slot / params.BeaconConfig().SlotsPerEpoch,
				"status": assignment.Status,
			}
			if assignment.Status == pb.ValidatorStatus_ACTIVE {
				if assignment.ProposerSlot > 0 {
					lFields["proposerSlot"] = assignment.ProposerSlot
				}
				lFields["attesterSlot"] = assignment.AttesterSlot
			}
			log.WithFields(lFields).Info("New assignment")
		}
	}

	return nil
}

// RolesAt slot returns the validator roles at the given slot. Returns nil if the
// validator is known to not have a roles at the at slot. Returns UNKNOWN if the
// validator assignments are unknown. Otherwise returns a valid ValidatorRole map.
func (v *validator) RolesAt(slot uint64) map[[48]byte]pb.ValidatorRole {
	rolesAt := make(map[[48]byte]pb.ValidatorRole)
	for _, assignment := range v.assignments.ValidatorAssignment {
		var role pb.ValidatorRole
		switch {
		case assignment == nil:
			role = pb.ValidatorRole_UNKNOWN
		case assignment.ProposerSlot == slot && assignment.AttesterSlot == slot:
			role = pb.ValidatorRole_BOTH
		case assignment.AttesterSlot == slot:
			role = pb.ValidatorRole_ATTESTER
		case assignment.ProposerSlot == slot:
			role = pb.ValidatorRole_PROPOSER
		default:
			role = pb.ValidatorRole_UNKNOWN
		}
		var pubKey [48]byte
		copy(pubKey[:], assignment.PublicKey)
		rolesAt[pubKey] = role
	}
	return rolesAt
}
