package rpc

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidatorServer defines a server implementation of the gRPC Validator service,
// providing RPC endpoints for obtaining validator assignments per epoch, the slots
// and shards in which particular validators need to perform their responsibilities,
// and more.
type ValidatorServer struct {
	ctx                context.Context
	beaconDB           db.Database
	headFetcher        blockchain.HeadFetcher
	canonicalStateChan chan *pbp2p.BeaconState
	blockFetcher       powchain.POWBlockFetcher
	depositFetcher     depositcache.DepositFetcher
	chainStartFetcher  powchain.ChainStartFetcher
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
			return errors.New("stream context closed, exiting gorutine")
		case <-vs.ctx.Done():
			return errors.New("rpc context closed, exiting goroutine")
		}
	}
}

// ValidatorIndex is called by a validator to get its index location that corresponds
// to the attestation bit fields.
func (vs *ValidatorServer) ValidatorIndex(ctx context.Context, req *pb.ValidatorIndexRequest) (*pb.ValidatorIndexResponse, error) {
	index, ok, err := vs.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(req.PublicKey))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.Internal, "could not find validator index for public key  %#x not found", req.PublicKey)
	}

	return &pb.ValidatorIndexResponse{Index: uint64(index)}, nil
}

// ValidatorPerformance reports the validator's latest balance along with other important metrics on
// rewards and penalties throughout its lifecycle in the beacon chain.
func (vs *ValidatorServer) ValidatorPerformance(
	ctx context.Context, req *pb.ValidatorPerformanceRequest,
) (*pb.ValidatorPerformanceResponse, error) {
	index, ok, err := vs.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(req.PublicKey))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.Internal, "could not find  validator index for public key  %#x not found", req.PublicKey)
	}
	headState := vs.headFetcher.HeadState()

	activeCount, err := helpers.ActiveValidatorCount(headState, helpers.SlotToEpoch(req.Slot))
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve active validator count")
	}

	totalActiveBalance, err := helpers.TotalActiveBalance(headState)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve active balance")
	}

	avgBalance := float32(totalActiveBalance / activeCount)
	balance := headState.Balances[index]
	return &pb.ValidatorPerformanceResponse{
		Balance:                       balance,
		AverageActiveValidatorBalance: avgBalance,
		TotalValidators:               uint64(len(headState.Validators)),
		TotalActiveValidators:         uint64(activeCount),
	}, nil
}

// CommitteeAssignment returns the committee assignment response from a given validator public key.
// The committee assignment response contains the following fields for the current and previous epoch:
//	1.) The list of validators in the committee.
//	2.) The shard to which the committee is assigned.
//	3.) The slot at which the committee is assigned.
//	4.) The bool signaling if the validator is expected to propose a block at the assigned slot.
func (vs *ValidatorServer) CommitteeAssignment(ctx context.Context, req *pb.AssignmentRequest) (*pb.AssignmentResponse, error) {
	var err error
	s := vs.headFetcher.HeadState()

	// Advance state with empty transitions up to the requested epoch start slot.
	if epochStartSlot := helpers.StartSlot(req.EpochStart); s.Slot < epochStartSlot {
		s, err = state.ProcessSlots(ctx, s, epochStartSlot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process slots up to %d", epochStartSlot)
		}
	}

	validatorIndexMap := stateutils.ValidatorIndexMap(s)
	var assignments []*pb.AssignmentResponse_ValidatorAssignment

	for _, pk := range req.PublicKeys {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		idx, ok := validatorIndexMap[bytesutil.ToBytes32(pk)]
		// Default assignment for every validator
		assignment := &pb.AssignmentResponse_ValidatorAssignment{
			PublicKey: pk,
			Status:    pb.ValidatorStatus_UNKNOWN_STATUS,
		}

		v := s.Validators[idx]
		// Update validator assignment when it is active
		if ok && helpers.IsActiveValidator(v, helpers.CurrentEpoch(s)) {
			assignment, err = vs.assignment(pk, s, req.EpochStart)
			if err != nil {
				return nil, err
			}
		} else if ok {
			// Update inactive validator's status
			status := vs.lookupValidatorStatus(uint64(idx), s)
			assignment.Status = status
		}
		assignments = append(assignments, assignment)
	}

	return &pb.AssignmentResponse{
		ValidatorAssignment: assignments,
	}, nil
}

func (vs *ValidatorServer) assignment(
	pubkey []byte,
	beaconState *pbp2p.BeaconState,
	epochStart uint64,
) (*pb.AssignmentResponse_ValidatorAssignment, error) {

	if len(pubkey) != params.BeaconConfig().BLSPubkeyLength {
		return nil, fmt.Errorf(
			"expected public key to have length %d, received %d",
			params.BeaconConfig().BLSPubkeyLength,
			len(pubkey),
		)
	}

	idx, ok, err := vs.beaconDB.ValidatorIndex(context.Background(), bytesutil.ToBytes48(pubkey))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.Internal, "could not find validator index for public key  %#x not found", pubkey)
	}

	committee, shard, slot, isProposer, err :=
		helpers.CommitteeAssignment(beaconState, epochStart, uint64(idx))
	if err != nil {
		return nil, err
	}
	status := vs.lookupValidatorStatus(idx, beaconState)
	return &pb.AssignmentResponse_ValidatorAssignment{
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
	headState := vs.headFetcher.HeadState()
	chainStarted := headState != nil
	chainStartDeposits := vs.chainStartFetcher.ChainStartDeposits()
	chainStartPubKeys := make(map[[48]byte]bool)
	for _, dep := range chainStartDeposits {
		chainStartPubKeys[bytesutil.ToBytes48(dep.Data.PublicKey)] = true
	}
	validatorIndexMap := stateutils.ValidatorIndexMap(headState)
	return vs.validatorStatus(ctx, req.PublicKey, chainStarted, chainStartPubKeys, validatorIndexMap, headState), nil
}

// MultipleValidatorStatus returns the validator status response for the set of validators
// requested by their pubkeys.
func (vs *ValidatorServer) MultipleValidatorStatus(
	ctx context.Context,
	pubkeys [][]byte) (bool, []*pb.ValidatorActivationResponse_Status, error) {
	hasChainStarted := vs.headFetcher.HeadState() != nil
	if !hasChainStarted {
		return false, nil, nil
	}
	activeValidatorExists := false
	statusResponses := make([]*pb.ValidatorActivationResponse_Status, len(pubkeys))
	headState := vs.headFetcher.HeadState()

	chainStartDeposits := vs.chainStartFetcher.ChainStartDeposits()
	chainStartPubKeys := make(map[[48]byte]bool)
	for _, dep := range chainStartDeposits {
		chainStartPubKeys[bytesutil.ToBytes48(dep.Data.PublicKey)] = true
	}
	validatorIndexMap := stateutils.ValidatorIndexMap(headState)
	for i, key := range pubkeys {
		if ctx.Err() != nil {
			return false, nil, ctx.Err()
		}
		status := vs.validatorStatus(ctx, key, hasChainStarted, chainStartPubKeys, validatorIndexMap, headState)
		if status == nil {
			continue
		}
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

// ExitedValidators queries validator statuses for a give list of validators
// and returns a filtered list of validator keys that are exited.
func (vs *ValidatorServer) ExitedValidators(
	ctx context.Context,
	req *pb.ExitedValidatorsRequest) (*pb.ExitedValidatorsResponse, error) {

	_, statuses, err := vs.MultipleValidatorStatus(ctx, req.PublicKeys)
	if err != nil {
		return nil, err
	}

	exitedKeys := make([][]byte, 0)
	for _, status := range statuses {
		s := status.Status.Status
		if s == pb.ValidatorStatus_EXITED ||
			s == pb.ValidatorStatus_EXITED_SLASHED ||
			s == pb.ValidatorStatus_INITIATED_EXIT {
			exitedKeys = append(exitedKeys, status.PublicKey)
		}
	}

	resp := &pb.ExitedValidatorsResponse{
		PublicKeys: exitedKeys,
	}

	return resp, nil
}

func (vs *ValidatorServer) validatorStatus(
	ctx context.Context, pubKey []byte, chainStarted bool,
	chainStartKeys map[[48]byte]bool, idxMap map[[32]byte]int,
	beaconState *pbp2p.BeaconState) *pb.ValidatorStatusResponse {
	pk := bytesutil.ToBytes32(pubKey)
	valIdx, ok := idxMap[pk]
	_, eth1BlockNumBigInt := vs.depositFetcher.DepositByPubkey(ctx, pubKey)
	if eth1BlockNumBigInt == nil {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch:        params.BeaconConfig().FarFutureEpoch,
			Eth1DepositBlockNumber: 0,
		}
	}

	if !ok {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch:        params.BeaconConfig().FarFutureEpoch,
			Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
		}
	}

	if !chainStarted {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch:        params.BeaconConfig().FarFutureEpoch,
			Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
		}
	}

	if exists := chainStartKeys[bytesutil.ToBytes48(pubKey)]; exists {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_ACTIVE,
			ActivationEpoch:        0,
			Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
			DepositInclusionSlot:   0,
		}
	}

	depositBlockSlot, err := vs.depositBlockSlot(ctx, beaconState.Slot, eth1BlockNumBigInt, beaconState)
	if err != nil {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch:        params.BeaconConfig().FarFutureEpoch,
			Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
		}
	}

	if depositBlockSlot == 0 {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch:        params.BeaconConfig().FarFutureEpoch,
			Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
		}
	}

	currEpoch := helpers.CurrentEpoch(beaconState)
	activationEpoch := params.BeaconConfig().FarFutureEpoch
	var validatorInState *ethpb.Validator
	var validatorIndex uint64
	for idx, val := range beaconState.Validators {
		if ctx.Err() != nil {
			return nil
		}

		if bytes.Equal(val.PublicKey, pubKey) {
			if helpers.IsActiveValidator(val, currEpoch) {
				activationEpoch = val.ActivationEpoch
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
		for j := len(beaconState.Validators) - 1; j >= 0; j-- {
			if helpers.IsActiveValidator(beaconState.Validators[j], currEpoch) {
				lastActivatedValidatorIdx = uint64(j)
				break
			}
		}
		// Our position in the activation queue is the above index - our validator index.
		positionInQueue = validatorIndex - lastActivatedValidatorIdx
	}

	status := vs.lookupValidatorStatus(uint64(valIdx), beaconState)
	return &pb.ValidatorStatusResponse{
		Status:                    status,
		Eth1DepositBlockNumber:    eth1BlockNumBigInt.Uint64(),
		PositionInActivationQueue: positionInQueue,
		DepositInclusionSlot:      depositBlockSlot,
		ActivationEpoch:           activationEpoch,
	}
}

func (vs *ValidatorServer) lookupValidatorStatus(validatorIdx uint64, beaconState *pbp2p.BeaconState) pb.ValidatorStatus {
	var status pb.ValidatorStatus
	v := beaconState.Validators[validatorIdx]
	epoch := helpers.CurrentEpoch(beaconState)
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	if epoch < v.ActivationEpoch {
		status = pb.ValidatorStatus_PENDING_ACTIVE
	} else if v.ExitEpoch == farFutureEpoch {
		status = pb.ValidatorStatus_ACTIVE
	} else if epoch >= v.WithdrawableEpoch {
		status = pb.ValidatorStatus_WITHDRAWABLE
	} else if v.Slashed && epoch >= v.ExitEpoch {
		status = pb.ValidatorStatus_EXITED_SLASHED
	} else if epoch >= v.ExitEpoch {
		status = pb.ValidatorStatus_EXITED
	} else if v.ExitEpoch != farFutureEpoch {
		status = pb.ValidatorStatus_INITIATED_EXIT
	} else {
		status = pb.ValidatorStatus_UNKNOWN_STATUS
	}

	return status
}

func (vs *ValidatorServer) depositBlockSlot(ctx context.Context, currentSlot uint64,
	eth1BlockNumBigInt *big.Int, beaconState *pbp2p.BeaconState) (uint64, error) {
	blockTimeStamp, err := vs.blockFetcher.BlockTimeByHeight(ctx, eth1BlockNumBigInt)
	if err != nil {
		return 0, err
	}
	followTime := time.Duration(params.BeaconConfig().Eth1FollowDistance*params.BeaconConfig().GoerliBlockTime) * time.Second
	eth1UnixTime := time.Unix(int64(blockTimeStamp), 0).Add(followTime)

	votingPeriodSlots := helpers.StartSlot(params.BeaconConfig().SlotsPerEth1VotingPeriod / params.BeaconConfig().SlotsPerEpoch)
	votingPeriodSeconds := time.Duration(votingPeriodSlots*params.BeaconConfig().SecondsPerSlot) * time.Second
	timeToInclusion := eth1UnixTime.Add(votingPeriodSeconds)

	eth2Genesis := time.Unix(int64(beaconState.GenesisTime), 0)
	eth2TimeDifference := timeToInclusion.Sub(eth2Genesis).Seconds()
	depositBlockSlot := uint64(eth2TimeDifference) / params.BeaconConfig().SecondsPerSlot

	if depositBlockSlot > currentSlot {
		return 0, nil
	}

	return depositBlockSlot, nil
}

// DomainData fetches the current domain version information from the beacon state.
func (vs *ValidatorServer) DomainData(ctx context.Context, request *pb.DomainRequest) (*pb.DomainResponse, error) {
	headState := vs.headFetcher.HeadState()
	dv := helpers.Domain(headState, request.Epoch, request.Domain)
	return &pb.DomainResponse{
		SignatureDomain: dv,
	}, nil
}
