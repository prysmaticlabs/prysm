// Package client represents the functionality to act as a validator.
package client

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

type validator struct {
	genesisTime          uint64
	ticker               *slotutil.SlotTicker
	db                   *db.Store
	duties               *ethpb.DutiesResponse
	validatorClient      ethpb.BeaconNodeValidatorClient
	beaconClient         ethpb.BeaconChainClient
	graffiti             []byte
	node                 ethpb.NodeClient
	keyManager           keymanager.KeyManager
	prevBalance          map[[48]byte]uint64
	logValidatorBalances bool
	emitAccountMetrics   bool
	attLogs              map[[32]byte]*attSubmitted
	attLogsLock          sync.Mutex
	domainDataLock       sync.Mutex
	domainDataCache      *ristretto.Cache
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
	validatingKeys, err := v.keyManager.FetchValidatingKeys()
	if err != nil {
		return errors.Wrap(err, "could not fetch validating keys")
	}
	req := &ethpb.ValidatorActivationRequest{
		PublicKeys: bytesutil.FromBytes48Array(validatingKeys),
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

func (v *validator) checkAndLogValidatorStatus(validatorStatuses []*ethpb.ValidatorActivationResponse_Status) [][]byte {
	var activatedKeys [][]byte
	for _, status := range validatorStatuses {
		log := log.WithFields(logrus.Fields{
			"pubKey": fmt.Sprintf("%#x", bytesutil.Trunc(status.PublicKey[:])),
			"status": status.Status.Status.String(),
		})
		if status.Status.Status == ethpb.ValidatorStatus_ACTIVE {
			activatedKeys = append(activatedKeys, status.PublicKey)
			continue
		}
		if status.Status.Status == ethpb.ValidatorStatus_EXITED {
			log.Info("Validator exited")
			continue
		}
		if status.Status.Status == ethpb.ValidatorStatus_DEPOSITED {
			log.WithField("expectedInclusionSlot", status.Status.DepositInclusionSlot).Info(
				"Deposit for validator received but not processed into state")
			continue
		}
		if uint64(status.Status.ActivationEpoch) == params.BeaconConfig().FarFutureEpoch {
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
	head, err := v.beaconClient.GetChainHead(ctx, &ptypes.Empty{})
	if err != nil {
		return 0, err
	}
	return head.HeadSlot, nil
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

// UpdateDuties checks the slot number to determine if the validator's
// list of upcoming assignments needs to be updated. For example, at the
// beginning of a new epoch.
func (v *validator) UpdateDuties(ctx context.Context, slot uint64) error {
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 && v.duties != nil {
		// Do nothing if not epoch start AND assignments already exist.
		return nil
	}
	// Set deadline to end of epoch.
	ctx, cancel := context.WithDeadline(ctx, v.SlotDeadline(helpers.StartSlot(helpers.SlotToEpoch(slot)+1)))
	defer cancel()
	ctx, span := trace.StartSpan(ctx, "validator.UpdateAssignments")
	defer span.End()

	validatingKeys, err := v.keyManager.FetchValidatingKeys()
	if err != nil {
		return err
	}
	req := &ethpb.DutiesRequest{
		Epoch:      slot / params.BeaconConfig().SlotsPerEpoch,
		PublicKeys: bytesutil.FromBytes48Array(validatingKeys),
	}

	resp, err := v.validatorClient.GetDuties(ctx, req)
	if err != nil {
		v.duties = nil // Clear assignments so we know to retry the request.
		log.Error(err)
		return err
	}

	v.duties = resp
	// Only log the full assignments output on epoch start to be less verbose.
	if slot%params.BeaconConfig().SlotsPerEpoch == 0 {
		for _, duty := range v.duties.Duties {
			lFields := logrus.Fields{
				"pubKey":         fmt.Sprintf("%#x", bytesutil.Trunc(duty.PublicKey)),
				"validatorIndex": duty.ValidatorIndex,
				"committeeIndex": duty.CommitteeIndex,
				"epoch":          slot / params.BeaconConfig().SlotsPerEpoch,
				"status":         duty.Status,
			}

			if duty.Status == ethpb.ValidatorStatus_ACTIVE {
				if duty.ProposerSlot > 0 {
					lFields["proposerSlot"] = duty.ProposerSlot
				}
				lFields["attesterSlot"] = duty.AttesterSlot
			}

			log.WithFields(lFields).Info("New assignment")
		}
	}

	return nil
}

// RolesAt slot returns the validator roles at the given slot. Returns nil if the
// validator is known to not have a roles at the at slot. Returns UNKNOWN if the
// validator assignments are unknown. Otherwise returns a valid ValidatorRole map.
func (v *validator) RolesAt(ctx context.Context, slot uint64) (map[[48]byte][]pb.ValidatorRole, error) {
	rolesAt := make(map[[48]byte][]pb.ValidatorRole)
	for _, duty := range v.duties.Duties {
		var roles []pb.ValidatorRole

		if duty == nil {
			continue
		}
		if duty.ProposerSlot > 0 && duty.ProposerSlot == slot {
			roles = append(roles, pb.ValidatorRole_PROPOSER)
		}
		if duty.AttesterSlot == slot {
			roles = append(roles, pb.ValidatorRole_ATTESTER)

			aggregator, err := v.isAggregator(ctx, duty.Committee, slot, bytesutil.ToBytes48(duty.PublicKey))
			if err != nil {
				return nil, errors.Wrap(err, "could not check if a validator is an aggregator")
			}
			if aggregator {
				roles = append(roles, pb.ValidatorRole_AGGREGATOR)
			}

		}
		if len(roles) == 0 {
			roles = append(roles, pb.ValidatorRole_UNKNOWN)
		}

		var pubKey [48]byte
		copy(pubKey[:], duty.PublicKey)
		rolesAt[pubKey] = roles
	}
	return rolesAt, nil
}

// isAggregator checks if a validator is an aggregator of a given slot, it uses the selection algorithm outlined in:
// https://github.com/ethereum/eth2.0-specs/blob/v0.9.3/specs/validator/0_beacon-chain-validator.md#aggregation-selection
func (v *validator) isAggregator(ctx context.Context, committee []uint64, slot uint64, pubKey [48]byte) (bool, error) {
	modulo := uint64(1)
	if len(committee)/int(params.BeaconConfig().TargetAggregatorsPerCommittee) > 1 {
		modulo = uint64(len(committee)) / params.BeaconConfig().TargetAggregatorsPerCommittee
	}

	slotSig, err := v.signSlot(ctx, pubKey, slot)
	if err != nil {
		return false, err
	}

	b := hashutil.Hash(slotSig)

	return binary.LittleEndian.Uint64(b[:8])%modulo == 0, nil
}

// UpdateDomainDataCaches by making calls for all of the possible domain data. These can change when
// the fork version changes which can happen once per epoch. Although changing for the fork version
// is very rare, a validator should check these data every epoch to be sure the validator is
// participating on the correct fork version.
func (v *validator) UpdateDomainDataCaches(ctx context.Context, slot uint64) {
	if !featureconfig.Get().EnableDomainDataCache {
		return
	}

	for _, d := range [][]byte{
		params.BeaconConfig().DomainRandao,
		params.BeaconConfig().DomainBeaconAttester,
		params.BeaconConfig().DomainBeaconProposer,
	} {
		_, err := v.domainData(ctx, helpers.SlotToEpoch(slot), d)
		if err != nil {
			log.WithError(err).Errorf("Failed to update domain data for domain %v", d)
		}
	}
}

func (v *validator) domainData(ctx context.Context, epoch uint64, domain []byte) (*ethpb.DomainResponse, error) {
	v.domainDataLock.Lock()
	defer v.domainDataLock.Unlock()

	req := &ethpb.DomainRequest{
		Epoch:  epoch,
		Domain: domain,
	}

	key := strings.Join([]string{strconv.FormatUint(req.Epoch, 10), hex.EncodeToString(req.Domain)}, ",")

	if featureconfig.Get().EnableDomainDataCache {
		if val, ok := v.domainDataCache.Get(key); ok {
			return proto.Clone(val.(proto.Message)).(*ethpb.DomainResponse), nil
		}
	}

	res, err := v.validatorClient.DomainData(ctx, req)
	if err != nil {
		return nil, err
	}

	if featureconfig.Get().EnableDomainDataCache {
		v.domainDataCache.Set(key, proto.Clone(res), 1)
	}

	return res, nil
}
