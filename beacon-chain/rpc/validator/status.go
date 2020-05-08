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
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errPubkeyDoesNotExist = errors.New("pubkey does not exist")

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
	req *ethpb.ValidatorStatusRequest) (*ethpb.ValidatorStatusResponse, error) {
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	return vs.validatorStatus(ctx, req.PublicKey, headState), nil
}

// multipleValidatorStatus returns the validator status response for the set of validators
// requested by their pub keys.
func (vs *Server) multipleValidatorStatus(
	ctx context.Context,
	pubkeys [][]byte,
) (bool, []*ethpb.ValidatorActivationResponse_Status, error) {
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return false, nil, err
	}
	activeValidatorExists := false
	statusResponses := make([]*ethpb.ValidatorActivationResponse_Status, len(pubkeys))
	for i, key := range pubkeys {
		if ctx.Err() != nil {
			return false, nil, ctx.Err()
		}
		status := vs.validatorStatus(ctx, key, headState)
		if status == nil {
			continue
		}
		resp := &ethpb.ValidatorActivationResponse_Status{
			Status:    status,
			PublicKey: key,
		}
		statusResponses[i] = resp
		if status.Status == ethpb.ValidatorStatus_ACTIVE {
			activeValidatorExists = true
		}
	}

	return activeValidatorExists, statusResponses, nil
}

func (vs *Server) validatorStatus(ctx context.Context, pubKey []byte, headState *stateTrie.BeaconState) *ethpb.ValidatorStatusResponse {
	ctx, span := trace.StartSpan(ctx, "validatorServer.validatorStatus")
	defer span.End()

	resp := &ethpb.ValidatorStatusResponse{
		Status:          ethpb.ValidatorStatus_UNKNOWN_STATUS,
		ActivationEpoch: params.BeaconConfig().FarFutureEpoch,
	}
	vStatus, idx, err := vs.retrieveStatusFromState(ctx, pubKey, headState)
	if err != nil && err != errPubkeyDoesNotExist {
		traceutil.AnnotateError(span, err)
		return resp
	}
	resp.Status = vStatus
	if err != errPubkeyDoesNotExist {
		val, err := headState.ValidatorAtIndexReadOnly(idx)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return resp
		}
		resp.ActivationEpoch = val.ActivationEpoch()
	}

	switch resp.Status {
	// Unknown status means the validator has not been put into the state yet.
	case ethpb.ValidatorStatus_UNKNOWN_STATUS:
		// If no connection to ETH1, the deposit block number or position in queue cannot be determined.
		if !vs.Eth1InfoFetcher.IsConnectedToETH1() {
			log.Warn("Not connected to ETH1. Cannot determine validator ETH1 deposit block number")
			return resp
		}
		_, eth1BlockNumBigInt := vs.DepositFetcher.DepositByPubkey(ctx, pubKey)
		if eth1BlockNumBigInt == nil { // No deposit found in ETH1.
			return resp
		}

		// Mark a validator as DEPOSITED if their deposit is visible.
		resp.Status = ethpb.ValidatorStatus_DEPOSITED

		resp.Eth1DepositBlockNumber = eth1BlockNumBigInt.Uint64()

		depositBlockSlot, err := vs.depositBlockSlot(ctx, eth1BlockNumBigInt, headState)
		if err != nil {
			return resp
		}
		resp.DepositInclusionSlot = depositBlockSlot
	// Deposited and Pending mean the validator has been put into the state.
	case ethpb.ValidatorStatus_DEPOSITED, ethpb.ValidatorStatus_PENDING:
		var lastActivatedValidatorIdx uint64
		for j := headState.NumValidators() - 1; j >= 0; j-- {
			val, err := headState.ValidatorAtIndexReadOnly(uint64(j))
			if err != nil {
				return resp
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
	default:
		return resp
	}

	return resp
}

func (vs *Server) retrieveStatusFromState(
	ctx context.Context,
	pubKey []byte,
	headState *stateTrie.BeaconState,
) (ethpb.ValidatorStatus, uint64, error) {
	if headState == nil {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS, 0, errors.New("head state does not exist")
	}
	idx, ok := headState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
	if !ok || int(idx) >= headState.NumValidators() {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS, 0, errPubkeyDoesNotExist
	}
	return vs.assignmentStatus(idx, headState), idx, nil
}

func (vs *Server) assignmentStatus(validatorIdx uint64, beaconState *stateTrie.BeaconState) ethpb.ValidatorStatus {
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

func (vs *Server) depositBlockSlot(ctx context.Context, eth1BlockNumBigInt *big.Int, beaconState *stateTrie.BeaconState) (uint64, error) {
	var depositBlockSlot uint64
	blockTimeStamp, err := vs.BlockFetcher.BlockTimeByHeight(ctx, eth1BlockNumBigInt)
	if err != nil {
		return 0, err
	}
	followTime := time.Duration(params.BeaconConfig().Eth1FollowDistance*params.BeaconConfig().GoerliBlockTime) * time.Second
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
