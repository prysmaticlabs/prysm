package rpc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ValidatorServer defines a server implementation of the gRPC Validator service,
// providing RPC endpoints for obtaining validator assignments per epoch, the slots
// and shards in which particular validators need to perform their responsibilities,
// and more.
type ValidatorServer struct {
	ctx                context.Context
	beaconDB           *db.BeaconDB
	chainService       chainService
	canonicalStateChan chan *pbp2p.BeaconState
	powChainService    powChainService
}

// WaitForActivation checks if a validator public key exists in the active validator registry of the current
// beacon state, if not, then it creates a stream which listens for canonical states which contain
// the validator with the public key as an active validator record.
func (vs *ValidatorServer) WaitForActivation(req *pb.ValidatorActivationRequest, stream pb.ValidatorService_WaitForActivationServer) error {
	activeValidatorExists, validatorStatuses, err := vs.MultipleValidatorStatus(stream.Context(), req.PublicKeys)
	if err != nil {
		return err
	}
	res := &pb.ValidatorActivationResponse{
		Statuses: validatorStatuses,
	}
	if activeValidatorExists {
		return stream.Send(res)
	}
	if err := stream.Send(res); err != nil {
		return err
	}

	for {
		select {
		case <-time.After(6 * time.Second):
			activeValidatorExists, validatorStatuses, err := vs.MultipleValidatorStatus(stream.Context(), req.PublicKeys)
			if err != nil {
				return err
			}
			res := &pb.ValidatorActivationResponse{
				Statuses: validatorStatuses,
			}
			if activeValidatorExists {
				return stream.Send(res)
			}
			if err := stream.Send(res); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return errors.New("stream context closed,exiting gorutine")
		case <-vs.ctx.Done():
			return errors.New("rpc context closed, exiting goroutine")
		}
	}
}

// ValidatorIndex is called by a validator to get its index location that corresponds
// to the attestation bit fields.
func (vs *ValidatorServer) ValidatorIndex(ctx context.Context, req *pb.ValidatorIndexRequest) (*pb.ValidatorIndexResponse, error) {
	index, err := vs.beaconDB.ValidatorIndex(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not get validator index: %v", err)
	}

	return &pb.ValidatorIndexResponse{Index: uint64(index)}, nil
}

// ValidatorPerformance reports the validator's latest balance along with other important metrics on
// rewards and penalties throughout its lifecycle in the beacon chain.
func (vs *ValidatorServer) ValidatorPerformance(
	ctx context.Context, req *pb.ValidatorPerformanceRequest,
) (*pb.ValidatorPerformanceResponse, error) {
	index, err := vs.beaconDB.ValidatorIndex(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not get validator index: %v", err)
	}
	validatorRegistry, err := vs.beaconDB.ValidatorRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve beacon state: %v", err)
	}
	activeIndices := helpers.ActiveValidatorIndices(validatorRegistry, helpers.SlotToEpoch(req.Slot))
	validatorBalances, err := vs.beaconDB.ValidatorBalances(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve validator balances %v", err)
	}
	totalActiveBalance := float32(0)
	for _, idx := range activeIndices {
		totalActiveBalance += float32(validatorBalances[idx])
	}
	avgBalance := totalActiveBalance / float32(len(activeIndices))
	balance := validatorBalances[index]
	return &pb.ValidatorPerformanceResponse{
		Balance:                       balance,
		AverageActiveValidatorBalance: avgBalance,
		TotalValidators:               uint64(len(validatorRegistry)),
		TotalActiveValidators:         uint64(len(activeIndices)),
	}, nil
}

// CommitteeAssignment returns the committee assignment response from a given validator public key.
// The committee assignment response contains the following fields for the current and previous epoch:
//	1.) The list of validators in the committee.
//	2.) The shard to which the committee is assigned.
//	3.) The slot at which the committee is assigned.
//	4.) The bool signaling if the validator is expected to propose a block at the assigned slot.
func (vs *ValidatorServer) CommitteeAssignment(
	ctx context.Context,
	req *pb.CommitteeAssignmentsRequest) (*pb.CommitteeAssignmentResponse, error) {
	beaconState, err := vs.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	chainHead, err := vs.beaconDB.ChainHead()
	if err != nil {
		return nil, fmt.Errorf("could not get chain head: %v", err)
	}
	headRoot, err := hashutil.HashBeaconBlock(chainHead)
	if err != nil {
		return nil, fmt.Errorf("could not hash block: %v", err)
	}

	for beaconState.Slot < req.EpochStart {
		beaconState, err = state.ExecuteStateTransition(
			ctx, beaconState, nil /* block */, headRoot, state.DefaultConfig(),
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute head transition: %v", err)
		}
	}

	var assignments []*pb.CommitteeAssignmentResponse_CommitteeAssignment
	activeKeys := vs.filterActivePublicKeys(beaconState, req.PublicKeys)
	for _, pk := range activeKeys {
		a, err := vs.assignment(pk, beaconState, req.EpochStart)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	assignments = vs.addNonActivePublicKeysAssignmentStatus(beaconState, req.PublicKeys, assignments)
	return &pb.CommitteeAssignmentResponse{
		Assignment: assignments,
	}, nil
}

func (vs *ValidatorServer) assignment(
	pubkey []byte,
	beaconState *pbp2p.BeaconState,
	epochStart uint64,
) (*pb.CommitteeAssignmentResponse_CommitteeAssignment, error) {

	if len(pubkey) != params.BeaconConfig().BLSPubkeyLength {
		return nil, fmt.Errorf(
			"expected public key to have length %d, received %d",
			params.BeaconConfig().BLSPubkeyLength,
			len(pubkey),
		)
	}

	idx, err := vs.beaconDB.ValidatorIndex(pubkey)
	if err != nil {
		return nil, fmt.Errorf("could not get active validator index: %v", err)
	}

	committee, shard, slot, isProposer, err :=
		helpers.CommitteeAssignment(beaconState, epochStart, uint64(idx), false)
	if err != nil {
		return nil, err
	}
	status := vs.lookupValidatorStatusFlag(idx, beaconState)
	return &pb.CommitteeAssignmentResponse_CommitteeAssignment{
		Committee:  committee,
		Shard:      shard,
		Slot:       slot,
		IsProposer: isProposer,
		PublicKey:  pubkey,
		Status:     status,
	}, nil
}

// ValidatorStatus returns the validator status of the current epoch.
// The status response can be one of the following:
//	PENDING_ACTIVE - validator is waiting to get activated.
//	ACTIVE - validator is active.
//	INITIATED_EXIT - validator has initiated an an exit request.
//	WITHDRAWABLE - validator's deposit can be withdrawn after lock up period.
//	EXITED - validator has exited, means the deposit has been withdrawn.
//	EXITED_SLASHED - validator was forcefully exited due to slashing.
func (vs *ValidatorServer) ValidatorStatus(
	ctx context.Context,
	req *pb.ValidatorIndexRequest) (*pb.ValidatorStatusResponse, error) {
	beaconState, err := vs.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	validatorIndexMap := stateutils.ValidatorIndexMap(beaconState)
	return vs.validatorStatus(ctx, req.PublicKey, validatorIndexMap, beaconState), nil
}

// MultipleValidatorStatus returns the validator status response for the set of validators
// requested by their pubkeys.
func (vs *ValidatorServer) MultipleValidatorStatus(
	ctx context.Context,
	pubkeys [][]byte) (bool, []*pb.ValidatorActivationResponse_Status, error) {
	activeValidatorExists := false
	statusResponses := make([]*pb.ValidatorActivationResponse_Status, len(pubkeys))
	beaconState, err := vs.beaconDB.HeadState(ctx)
	if err != nil {
		return false, nil, err
	}
	validatorIndexMap := stateutils.ValidatorIndexMap(beaconState)
	for i, key := range pubkeys {
		status := vs.validatorStatus(ctx, key, validatorIndexMap, beaconState)
		resp := &pb.ValidatorActivationResponse_Status{
			Status:    status,
			PublicKey: key,
		}
		statusResponses[i] = resp
		if status.Status == pb.ValidatorStatus_ACTIVE {
			activeValidatorExists = true
		}
	}

	return activeValidatorExists, statusResponses, nil
}

func (vs *ValidatorServer) validatorStatus(
	ctx context.Context, pubKey []byte, idxMap map[[32]byte]int, beaconState *pbp2p.BeaconState,
) *pb.ValidatorStatusResponse {
	pk := bytesutil.ToBytes32(pubKey)
	valIdx, ok := idxMap[pk]
	_, eth1BlockNumBigInt := vs.beaconDB.DepositByPubkey(ctx, pubKey)
	if eth1BlockNumBigInt == nil {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch:        params.BeaconConfig().FarFutureEpoch - params.BeaconConfig().GenesisEpoch,
			Eth1DepositBlockNumber: 0,
		}
	}

	if !ok {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch:        params.BeaconConfig().FarFutureEpoch - params.BeaconConfig().GenesisEpoch,
			Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
		}
	}

	depositBlockSlot, err := vs.depositBlockSlot(ctx, beaconState.Slot, eth1BlockNumBigInt, beaconState)
	if err != nil {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch:        params.BeaconConfig().FarFutureEpoch - params.BeaconConfig().GenesisEpoch,
			Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
		}
	}

	if depositBlockSlot == 0 {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch:        params.BeaconConfig().FarFutureEpoch - params.BeaconConfig().GenesisEpoch,
			Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
		}
	}

	currEpoch := helpers.CurrentEpoch(beaconState)
	var validatorInState *pbp2p.Validator
	var validatorIndex uint64
	for idx, val := range beaconState.ValidatorRegistry {
		if bytes.Equal(val.Pubkey, pubKey) {
			if helpers.IsActiveValidator(val, currEpoch) {
				return &pb.ValidatorStatusResponse{
					Status:                 pb.ValidatorStatus_ACTIVE,
					ActivationEpoch:        val.ActivationEpoch - params.BeaconConfig().GenesisEpoch,
					Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
					DepositInclusionSlot:   depositBlockSlot,
				}
			}
			validatorInState = val
			validatorIndex = uint64(idx)
			break
		}
	}

	var positionInQueue uint64
	// If the validator has deposited and has been added to the state:
	if validatorInState != nil {
		var lastActivatedValidatorIdx uint64
		for j := len(beaconState.ValidatorRegistry) - 1; j >= 0; j-- {
			if helpers.IsActiveValidator(beaconState.ValidatorRegistry[j], currEpoch) {
				lastActivatedValidatorIdx = uint64(j)
				break
			}
		}
		// Our position in the activation queue is the above index - our validator index.
		positionInQueue = validatorIndex - lastActivatedValidatorIdx
	}

	status := vs.lookupValidatorStatusFlag(uint64(valIdx), beaconState)
	return &pb.ValidatorStatusResponse{
		Status:                    status,
		Eth1DepositBlockNumber:    eth1BlockNumBigInt.Uint64(),
		PositionInActivationQueue: positionInQueue,
		DepositInclusionSlot:      depositBlockSlot,
		ActivationEpoch:           params.BeaconConfig().FarFutureEpoch - params.BeaconConfig().GenesisEpoch,
	}
}

func (vs *ValidatorServer) lookupValidatorStatusFlag(validatorIdx uint64, beaconState *pbp2p.BeaconState) pb.ValidatorStatus {
	var status pb.ValidatorStatus
	v := beaconState.ValidatorRegistry[validatorIdx]
	epoch := helpers.CurrentEpoch(beaconState)

	if v.ActivationEpoch > epoch {
		status = pb.ValidatorStatus_PENDING_ACTIVE
	} else if epoch >= v.ActivationEpoch && epoch < v.ExitEpoch {
		status = pb.ValidatorStatus_ACTIVE
	} else if v.StatusFlags == pbp2p.Validator_INITIATED_EXIT {
		status = pb.ValidatorStatus_INITIATED_EXIT
	} else if v.StatusFlags == pbp2p.Validator_WITHDRAWABLE {
		status = pb.ValidatorStatus_WITHDRAWABLE
	} else if epoch >= v.ExitEpoch && epoch >= v.SlashedEpoch {
		status = pb.ValidatorStatus_EXITED_SLASHED
	} else if epoch >= v.ExitEpoch {
		status = pb.ValidatorStatus_EXITED
	} else {
		status = pb.ValidatorStatus_UNKNOWN_STATUS
	}

	return status
}

// filterActivePublicKeys takes a list of validator public keys and returns
// the list of active public keys from the given state.
func (vs *ValidatorServer) filterActivePublicKeys(beaconState *pbp2p.BeaconState, pubkeys [][]byte) [][]byte {
	// Generate a map for O(1) lookup of existence of pub keys in request.
	pkMap := make(map[string]bool)
	for _, pk := range pubkeys {
		pkMap[hex.EncodeToString(pk)] = true
	}

	var activeKeys [][]byte
	currentEpoch := helpers.SlotToEpoch(beaconState.Slot)
	for _, v := range beaconState.ValidatorRegistry {
		if pkMap[hex.EncodeToString(v.Pubkey)] && helpers.IsActiveValidator(v, currentEpoch) {
			activeKeys = append(activeKeys, v.Pubkey)
		}
	}

	return activeKeys
}

func (vs *ValidatorServer) addNonActivePublicKeysAssignmentStatus(
	beaconState *pbp2p.BeaconState,
	pubkeys [][]byte,
	assignments []*pb.CommitteeAssignmentResponse_CommitteeAssignment,
) []*pb.CommitteeAssignmentResponse_CommitteeAssignment {
	// Generate a map for O(1) lookup of existence of pub keys in request.
	validatorMap := stateutils.ValidatorIndexMap(beaconState)
	currentEpoch := helpers.CurrentEpoch(beaconState)
	for _, pk := range pubkeys {
		hexPk := bytesutil.ToBytes32(pk)
		if valIdx, ok := validatorMap[hexPk]; !ok || !helpers.IsActiveValidator(beaconState.ValidatorRegistry[validatorMap[hexPk]], currentEpoch) {
			status := vs.lookupValidatorStatusFlag(uint64(valIdx), beaconState) //nolint:gosec
			if !ok {
				status = pb.ValidatorStatus_UNKNOWN_STATUS
			}
			a := &pb.CommitteeAssignmentResponse_CommitteeAssignment{
				PublicKey: pk,
				Status:    status,
			}
			assignments = append(assignments, a)
		}
	}
	return assignments
}

func (vs *ValidatorServer) depositBlockSlot(ctx context.Context, currentSlot uint64,
	eth1BlockNumBigInt *big.Int, beaconState *pbp2p.BeaconState) (uint64, error) {
	blockTimeStamp, err := vs.powChainService.BlockTimeByHeight(ctx, eth1BlockNumBigInt)
	if err != nil {
		return 0, err
	}
	followTime := time.Duration(params.BeaconConfig().Eth1FollowDistance*params.BeaconConfig().GoerliBlockTime) * time.Second
	eth1UnixTime := time.Unix(int64(blockTimeStamp), 0).Add(followTime)

	votingPeriodSlots := helpers.StartSlot(params.BeaconConfig().EpochsPerEth1VotingPeriod)
	votingPeriodSeconds := time.Duration(votingPeriodSlots*params.BeaconConfig().SecondsPerSlot) * time.Second
	timeToInclusion := eth1UnixTime.Add(votingPeriodSeconds)

	eth2Genesis := time.Unix(int64(beaconState.GenesisTime), 0)
	eth2TimeDifference := timeToInclusion.Sub(eth2Genesis).Seconds()
	depositBlockSlot := uint64(eth2TimeDifference) / params.BeaconConfig().SecondsPerSlot

	if depositBlockSlot > currentSlot-params.BeaconConfig().GenesisSlot {
		return 0, nil
	}

	return depositBlockSlot, nil
}
