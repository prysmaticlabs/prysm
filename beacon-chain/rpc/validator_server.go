package rpc

import (
	"context"
	"math/big"
	"time"

	ptypes "github.com/gogo/protobuf/types"
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
	stateFeedListener  blockchain.ChainFeeds
	chainStartChan     chan time.Time
}

// WaitForActivation checks if a validator public key exists in the active validator registry of the current
// beacon state, if not, then it creates a stream which listens for canonical states which contain
// the validator with the public key as an active validator record.
func (vs *ValidatorServer) WaitForActivation(req *pb.ValidatorActivationRequest, stream pb.ValidatorService_WaitForActivationServer) error {
	activeValidatorExists, validatorStatuses, err := vs.multipleValidatorStatus(stream.Context(), req.PublicKeys)
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
			activeValidatorExists, validatorStatuses, err := vs.multipleValidatorStatus(stream.Context(), req.PublicKeys)
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

// ValidatorIndex is called by a validator to get its index location in the beacon state.
func (vs *ValidatorServer) ValidatorIndex(ctx context.Context, req *pb.ValidatorIndexRequest) (*pb.ValidatorIndexResponse, error) {
	index, ok, err := vs.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(req.PublicKey))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.Internal, "could not find validator index for public key %#x not found", req.PublicKey)
	}

	return &pb.ValidatorIndexResponse{Index: uint64(index)}, nil
}

// ValidatorPerformance reports the validator's latest balance along with other important metrics on
// rewards and penalties throughout its lifecycle in the beacon chain.
func (vs *ValidatorServer) ValidatorPerformance(
	ctx context.Context, req *pb.ValidatorPerformanceRequest,
) (*pb.ValidatorPerformanceResponse, error) {
	var err error
	headState := vs.headFetcher.HeadState()
	// Advance state with empty transitions up to the requested epoch start slot.
	if req.Slot > headState.Slot {
		headState, err = state.ProcessSlots(ctx, headState, req.Slot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process slots up to %d", req.Slot)
		}
	}

	balances := make([]uint64, len(req.PublicKeys))
	missingValidators := make([][]byte, 0)
	for i, key := range req.PublicKeys {
		index, ok, err := vs.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(key))
		if err != nil || !ok {
			missingValidators = append(missingValidators, key)
			balances[i] = 0
			continue
		}
		balances[i] = headState.Balances[index]
	}

	activeCount, err := helpers.ActiveValidatorCount(headState, helpers.SlotToEpoch(req.Slot))
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve active validator count")
	}

	totalActiveBalance, err := helpers.TotalActiveBalance(headState)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve active balance")
	}

	avgBalance := float32(totalActiveBalance / activeCount)
	return &pb.ValidatorPerformanceResponse{
		Balances:                      balances,
		AverageActiveValidatorBalance: avgBalance,
		MissingValidators:             missingValidators,
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

	for _, pubKey := range req.PublicKeys {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Default assignment
		assignment := &pb.AssignmentResponse_ValidatorAssignment{
			PublicKey: pubKey,
			Status:    pb.ValidatorStatus_UNKNOWN_STATUS,
		}

		idx, ok := validatorIndexMap[bytesutil.ToBytes32(pubKey)]
		if ok {
			status := vs.assignmentStatus(uint64(idx), s)
			assignment.Status = status
			if status == pb.ValidatorStatus_ACTIVE {
				assignment, err = vs.assignment(uint64(idx), s, req.EpochStart)
				if err != nil {
					return nil, err
				}
				assignment.PublicKey = pubKey
			}
		}
		assignments = append(assignments, assignment)
	}

	return &pb.AssignmentResponse{
		ValidatorAssignment: assignments,
	}, nil
}

func (vs *ValidatorServer) assignment(idx uint64, beaconState *pbp2p.BeaconState, epoch uint64) (*pb.AssignmentResponse_ValidatorAssignment, error) {
	committee, shard, slot, isProposer, err := helpers.CommitteeAssignment(beaconState, epoch, idx)
	if err != nil {
		return nil, err
	}
	status := vs.assignmentStatus(idx, beaconState)
	return &pb.AssignmentResponse_ValidatorAssignment{
		Committee:  committee,
		Shard:      shard,
		Slot:       slot,
		IsProposer: isProposer,
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

	validatorIndexMap := stateutils.ValidatorIndexMap(headState)
	return vs.validatorStatus(ctx, req.PublicKey, validatorIndexMap, headState), nil
}

// multipleValidatorStatus returns the validator status response for the set of validators
// requested by their pub keys.
func (vs *ValidatorServer) multipleValidatorStatus(
	ctx context.Context,
	pubkeys [][]byte) (bool, []*pb.ValidatorActivationResponse_Status, error) {
	if vs.headFetcher.HeadState() == nil {
		return false, nil, nil
	}

	activeValidatorExists := false
	statusResponses := make([]*pb.ValidatorActivationResponse_Status, len(pubkeys))
	headState := vs.headFetcher.HeadState()

	validatorIndexMap := stateutils.ValidatorIndexMap(headState)
	for i, key := range pubkeys {
		if ctx.Err() != nil {
			return false, nil, ctx.Err()
		}
		status := vs.validatorStatus(ctx, key, validatorIndexMap, headState)
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

	_, statuses, err := vs.multipleValidatorStatus(ctx, req.PublicKeys)
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

// DomainData fetches the current domain version information from the beacon state.
func (vs *ValidatorServer) DomainData(ctx context.Context, request *pb.DomainRequest) (*pb.DomainResponse, error) {
	headState := vs.headFetcher.HeadState()
	dv := helpers.Domain(headState, request.Epoch, request.Domain)
	return &pb.DomainResponse{
		SignatureDomain: dv,
	}, nil
}

func (vs *ValidatorServer) validatorStatus(ctx context.Context, pubKey []byte, idxMap map[[32]byte]int, headState *pbp2p.BeaconState) *pb.ValidatorStatusResponse {
	_, eth1BlockNumBigInt := vs.depositFetcher.DepositByPubkey(ctx, pubKey)
	if eth1BlockNumBigInt == nil {
		return &pb.ValidatorStatusResponse{
			Status:          pb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	defaultUnknownResponse := &pb.ValidatorStatusResponse{
		Status:                 pb.ValidatorStatus_UNKNOWN_STATUS,
		ActivationEpoch:        params.BeaconConfig().FarFutureEpoch,
		Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
	}

	idx, ok := idxMap[bytesutil.ToBytes32(pubKey)]
	if !ok || headState == nil {
		return defaultUnknownResponse
	}

	depositBlockSlot, err := vs.depositBlockSlot(ctx, headState.Slot, eth1BlockNumBigInt, headState)
	if err != nil {
		return defaultUnknownResponse
	}

	if helpers.IsActiveValidator(headState.Validators[idx], headState.Slot) {
		return &pb.ValidatorStatusResponse{
			Status:                 pb.ValidatorStatus_ACTIVE,
			Eth1DepositBlockNumber: eth1BlockNumBigInt.Uint64(),
			ActivationEpoch:        headState.Validators[idx].ActivationEpoch,
			DepositInclusionSlot:   depositBlockSlot,
		}
	}

	var queuePosition uint64
	var lastActivatedValidatorIdx uint64
	for j := len(headState.Validators) - 1; j >= 0; j-- {
		if helpers.IsActiveValidator(headState.Validators[j], helpers.CurrentEpoch(headState)) {
			lastActivatedValidatorIdx = uint64(j)
			break
		}
	}
	// Our position in the activation queue is the above index - our validator index.
	queuePosition = uint64(idx) - lastActivatedValidatorIdx
	status := vs.assignmentStatus(uint64(idx), headState)
	return &pb.ValidatorStatusResponse{
		Status:                    status,
		Eth1DepositBlockNumber:    eth1BlockNumBigInt.Uint64(),
		PositionInActivationQueue: queuePosition,
		DepositInclusionSlot:      depositBlockSlot,
		ActivationEpoch:           headState.Validators[idx].ActivationEpoch,
	}
}

func (vs *ValidatorServer) assignmentStatus(validatorIdx uint64, beaconState *pbp2p.BeaconState) pb.ValidatorStatus {
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

// CanonicalHead of the current beacon chain. This method is requested on-demand
// by a validator when it is their time to propose or attest.
func (vs *ValidatorServer) CanonicalHead(ctx context.Context, req *ptypes.Empty) (*ethpb.BeaconBlock, error) {
	return vs.headFetcher.HeadBlock(), nil
}

// WaitForChainStart queries the logs of the Deposit Contract in order to verify the beacon chain
// has started its runtime and validators begin their responsibilities. If it has not, it then
// subscribes to an event stream triggered by the powchain service whenever the ChainStart log does
// occur in the Deposit Contract on ETH 1.0.
func (vs *ValidatorServer) WaitForChainStart(req *ptypes.Empty, stream pb.ValidatorService_WaitForChainStartServer) error {
	head, err := vs.beaconDB.HeadState(context.Background())
	if err != nil {
		return err
	}
	if head != nil {
		res := &pb.ChainStartResponse{
			Started:     true,
			GenesisTime: head.GenesisTime,
		}
		return stream.Send(res)
	}

	sub := vs.stateFeedListener.StateInitializedFeed().Subscribe(vs.chainStartChan)
	defer sub.Unsubscribe()
	for {
		select {
		case chainStartTime := <-vs.chainStartChan:
			log.Info("Sending ChainStart log and genesis time to connected validator clients")
			res := &pb.ChainStartResponse{
				Started:     true,
				GenesisTime: uint64(chainStartTime.Unix()),
			}
			return stream.Send(res)
		case <-sub.Err():
			return errors.New("subscriber closed, exiting goroutine")
		case <-vs.ctx.Done():
			return errors.New("rpc context closed, exiting goroutine")
		}
	}
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
