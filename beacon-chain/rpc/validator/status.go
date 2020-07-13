package validator

import (
	"context"
	"errors"
	"math/big"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errPubkeyDoesNotExist = errors.New("pubkey does not exist")
var nonExistentIndex = ^uint64(0)

// ValidatorStatus returns the validator status of the current epoch.
// The status response can be one of the following:
//  DEPOSITED - validator's deposit has been recognized by Ethereum 1, not yet recognized by Ethereum 2.
//  PENDING - validator is in Ethereum 2's activation queue.
//  ACTIVE - validator is active.
//  EXITING - validator has initiated an an exit request, or has dropped below the ejection balance and is being kicked out.
//  EXITED - validator is no longer validating.
//  SLASHING - validator has been kicked out due to meeting a slashing condition.
//  UNKNOWN_STATUS - validator does not have a known status in the network.
func (vs *Server) ValidatorStatus(
	ctx context.Context,
	req *ethpb.ValidatorStatusRequest,
) (*ethpb.ValidatorStatusResponse, error) {
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	vStatus, _ := vs.validatorStatus(ctx, headState, req.PublicKey)
	return vStatus, nil
}

// MultipleValidatorStatus is the same as ValidatorStatus. Supports retrieval of multiple
// validator statuses. Takes a list of public keys or a list of validator indices.
func (vs *Server) MultipleValidatorStatus(
	ctx context.Context,
	req *ethpb.MultipleValidatorStatusRequest,
) (*ethpb.MultipleValidatorStatusResponse, error) {
	if vs.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	responseCap := len(req.PublicKeys) + len(req.Indices)
	pubKeys := make([][]byte, 0, responseCap)
	filtered := make(map[[48]byte]bool)
	filtered[[48]byte{}] = true // Filter out keys with all zeros.
	// Filter out duplicate public keys.
	for _, pubKey := range req.PublicKeys {
		pubkeyBytes := bytesutil.ToBytes48(pubKey)
		if !filtered[pubkeyBytes] {
			pubKeys = append(pubKeys, pubKey)
			filtered[pubkeyBytes] = true
		}
	}
	// Convert indices to public keys.
	for _, idx := range req.Indices {
		pubkeyBytes := headState.PubkeyAtIndex(uint64(idx))
		if !filtered[pubkeyBytes] {
			pubKeys = append(pubKeys, pubkeyBytes[:])
			filtered[pubkeyBytes] = true
		}
	}
	// Fetch statuses from beacon state.
	statuses := make([]*ethpb.ValidatorStatusResponse, len(pubKeys))
	indices := make([]uint64, len(pubKeys))
	for i, pubKey := range pubKeys {
		statuses[i], indices[i] = vs.validatorStatus(ctx, headState, pubKey)
	}

	return &ethpb.MultipleValidatorStatusResponse{
		PublicKeys: pubKeys,
		Statuses:   statuses,
		Indices:    indices,
	}, nil
}

// activationStatus returns the validator status response for the set of validators
// requested by their pub keys.
func (vs *Server) activationStatus(
	ctx context.Context,
	pubKeys [][]byte,
) (bool, []*ethpb.ValidatorActivationResponse_Status, error) {
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return false, nil, err
	}
	activeValidatorExists := false
	statusResponses := make([]*ethpb.ValidatorActivationResponse_Status, len(pubKeys))
	for i, pubKey := range pubKeys {
		if ctx.Err() != nil {
			return false, nil, ctx.Err()
		}
		vStatus, idx := vs.validatorStatus(ctx, headState, pubKey)
		if vStatus == nil {
			continue
		}
		resp := &ethpb.ValidatorActivationResponse_Status{
			Status:    vStatus,
			PublicKey: pubKey,
			Index:     idx,
		}
		statusResponses[i] = resp
		if vStatus.Status == ethpb.ValidatorStatus_ACTIVE {
			activeValidatorExists = true
		}
	}

	return activeValidatorExists, statusResponses, nil
}

// validatorStatus searches for the requested validator's state and deposit to retrieve its inclusion estimate. Also returns the validators index.
func (vs *Server) validatorStatus(
	ctx context.Context,
	headState *stateTrie.BeaconState,
	pubKey []byte,
) (*ethpb.ValidatorStatusResponse, uint64) {
	ctx, span := trace.StartSpan(ctx, "ValidatorServer.validatorStatus")
	defer span.End()

	// Using ^0 as the default value for index, in case the validators index cannot be determined.
	resp := &ethpb.ValidatorStatusResponse{
		Status:          ethpb.ValidatorStatus_UNKNOWN_STATUS,
		ActivationEpoch: params.BeaconConfig().FarFutureEpoch,
	}
	vStatus, idx, err := retrieveStatusForPubKey(headState, pubKey)
	if err != nil && err != errPubkeyDoesNotExist {
		traceutil.AnnotateError(span, err)
		return resp, nonExistentIndex
	}
	resp.Status = vStatus
	if err != errPubkeyDoesNotExist {
		val, err := headState.ValidatorAtIndexReadOnly(idx)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return resp, idx
		}
		resp.ActivationEpoch = val.ActivationEpoch()
	}

	switch resp.Status {
	// Unknown status means the validator has not been put into the state yet.
	case ethpb.ValidatorStatus_UNKNOWN_STATUS:
		// If no connection to ETH1, the deposit block number or position in queue cannot be determined.
		if !vs.Eth1InfoFetcher.IsConnectedToETH1() {
			log.Warn("Not connected to ETH1. Cannot determine validator ETH1 deposit block number")
			return resp, nonExistentIndex
		}
		deposit, eth1BlockNumBigInt := vs.DepositFetcher.DepositByPubkey(ctx, pubKey)
		if eth1BlockNumBigInt == nil { // No deposit found in ETH1.
			return resp, nonExistentIndex
		}
		err := depositutil.VerifyDepositSignature(deposit.Data)
		if err != nil {
			resp.Status = ethpb.ValidatorStatus_INVALID
			return resp, nonExistentIndex
		}
		// Mark a validator as DEPOSITED if their deposit is visible.
		resp.Status = ethpb.ValidatorStatus_DEPOSITED

		resp.Eth1DepositBlockNumber = eth1BlockNumBigInt.Uint64()

		depositBlockSlot, err := vs.depositBlockSlot(ctx, headState, eth1BlockNumBigInt)
		if err != nil {
			return resp, nonExistentIndex
		}
		resp.DepositInclusionSlot = depositBlockSlot
		return resp, nonExistentIndex
	// Deposited and Pending mean the validator has been put into the state.
	case ethpb.ValidatorStatus_DEPOSITED, ethpb.ValidatorStatus_PENDING:
		var lastActivatedValidatorIdx uint64
		for j := headState.NumValidators() - 1; j >= 0; j-- {
			val, err := headState.ValidatorAtIndexReadOnly(uint64(j))
			if err != nil {
				return resp, idx
			}
			if helpers.IsActiveValidatorUsingTrie(val, helpers.CurrentEpoch(headState)) {
				lastActivatedValidatorIdx = uint64(j)
				break
			}
		}
		// Our position in the activation queue is the above index - our validator index.
		if lastActivatedValidatorIdx < idx {
			resp.PositionInActivationQueue = idx - lastActivatedValidatorIdx
		}
		return resp, idx
	default:
		return resp, idx
	}
}

func retrieveStatusForPubKey(headState *stateTrie.BeaconState, pubKey []byte) (ethpb.ValidatorStatus, uint64, error) {
	if headState == nil {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS, 0, errors.New("head state does not exist")
	}
	idx, ok := headState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
	if !ok || idx >= uint64(headState.NumValidators()) {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS, 0, errPubkeyDoesNotExist
	}
	return assignmentStatus(headState, idx), idx, nil
}

func assignmentStatus(beaconState *stateTrie.BeaconState, validatorIdx uint64) ethpb.ValidatorStatus {
	validator, err := beaconState.ValidatorAtIndexReadOnly(validatorIdx)
	if err != nil {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS
	}
	currentEpoch := helpers.CurrentEpoch(beaconState)
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	if validator == nil {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS
	}
	if currentEpoch < validator.ActivationEligibilityEpoch() {
		return ethpb.ValidatorStatus_DEPOSITED
	}
	if currentEpoch < validator.ActivationEpoch() {
		return ethpb.ValidatorStatus_PENDING
	}
	if validator.ExitEpoch() == farFutureEpoch {
		return ethpb.ValidatorStatus_ACTIVE
	}
	if currentEpoch < validator.ExitEpoch() {
		if validator.Slashed() {
			return ethpb.ValidatorStatus_SLASHING
		}
		return ethpb.ValidatorStatus_EXITING
	}
	return ethpb.ValidatorStatus_EXITED
}

func (vs *Server) depositBlockSlot(ctx context.Context, beaconState *stateTrie.BeaconState, eth1BlockNumBigInt *big.Int) (uint64, error) {
	var depositBlockSlot uint64
	blockTimeStamp, err := vs.BlockFetcher.BlockTimeByHeight(ctx, eth1BlockNumBigInt)
	if err != nil {
		return 0, err
	}
	followTime := time.Duration(params.BeaconConfig().Eth1FollowDistance*params.BeaconConfig().SecondsPerETH1Block) * time.Second
	eth1UnixTime := time.Unix(int64(blockTimeStamp), 0).Add(followTime)
	period := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().EpochsPerEth1VotingPeriod
	votingPeriod := time.Duration(period*params.BeaconConfig().SecondsPerSlot) * time.Second
	timeToInclusion := eth1UnixTime.Add(votingPeriod)

	eth2Genesis := time.Unix(int64(beaconState.GenesisTime()), 0)

	if eth2Genesis.After(timeToInclusion) {
		depositBlockSlot = 0
	} else {
		eth2TimeDifference := timeToInclusion.Sub(eth2Genesis).Seconds()
		depositBlockSlot = uint64(eth2TimeDifference) / params.BeaconConfig().SecondsPerSlot
	}

	return depositBlockSlot, nil
}
