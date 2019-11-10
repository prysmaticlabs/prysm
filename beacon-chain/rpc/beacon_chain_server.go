package rpc

import (
	"context"
	"fmt"
	"sort"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BeaconChainServer defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
type BeaconChainServer struct {
	beaconDB            db.Database
	ctx                 context.Context
	chainStartFetcher   powchain.ChainStartFetcher
	headFetcher         blockchain.HeadFetcher
	finalizationFetcher blockchain.FinalizationFetcher
	stateFeedListener   blockchain.ChainFeeds
	pool                operations.Pool
	incomingAttestation chan *ethpb.Attestation
	canonicalStateChan  chan *pbp2p.BeaconState
	chainStartChan      chan time.Time
}

// sortableAttestations implements the Sort interface to sort attestations
// by slot as the canonical sorting attribute.
type sortableAttestations []*ethpb.Attestation

func (s sortableAttestations) Len() int      { return len(s) }
func (s sortableAttestations) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortableAttestations) Less(i, j int) bool {
	return s[i].Data.Slot < s[j].Data.Slot
}

// ListAttestations retrieves attestations by block root, slot, or epoch.
// Attestations are sorted by data slot by default.
//
// The server may return an empty list when no attestations match the given
// filter criteria. This RPC should not return NOT_FOUND. Only one filter
// criteria should be used.
func (bs *BeaconChainServer) ListAttestations(
	ctx context.Context, req *ethpb.ListAttestationsRequest,
) (*ethpb.ListAttestationsResponse, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}
	var atts []*ethpb.Attestation
	var err error
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListAttestationsRequest_HeadBlockRoot:
		atts, err = bs.beaconDB.Attestations(ctx, filters.NewFilter().SetHeadBlockRoot(q.HeadBlockRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch attestations: %v", err)
		}
	case *ethpb.ListAttestationsRequest_SourceEpoch:
		atts, err = bs.beaconDB.Attestations(ctx, filters.NewFilter().SetSourceEpoch(q.SourceEpoch))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch attestations: %v", err)
		}
	case *ethpb.ListAttestationsRequest_SourceRoot:
		atts, err = bs.beaconDB.Attestations(ctx, filters.NewFilter().SetSourceRoot(q.SourceRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch attestations: %v", err)
		}
	case *ethpb.ListAttestationsRequest_TargetEpoch:
		atts, err = bs.beaconDB.Attestations(ctx, filters.NewFilter().SetTargetEpoch(q.TargetEpoch))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch attestations: %v", err)
		}
	case *ethpb.ListAttestationsRequest_TargetRoot:
		atts, err = bs.beaconDB.Attestations(ctx, filters.NewFilter().SetTargetRoot(q.TargetRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch attestations: %v", err)
		}
	default:
		return nil, status.Errorf(codes.Internal, "could not fetch attestations: %v", err)
	}
	// We sort attestations according to the Sortable interface.
	sort.Sort(sortableAttestations(atts))
	numAttestations := len(atts)

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numAttestations)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not paginate attestations: %v", err)
	}
	return &ethpb.ListAttestationsResponse{
		Attestations:  atts[start:end],
		TotalSize:     int32(numAttestations),
		NextPageToken: nextPageToken,
	}, nil
}

// AttestationPool retrieves pending attestations.
//
// The server returns a list of attestations that have been seen but not
// yet processed. Pool attestations eventually expire as the slot
// advances, so an attestation missing from this request does not imply
// that it was included in a block. The attestation may have expired.
// Refer to the ethereum 2.0 specification for more details on how
// attestations are processed and when they are no longer valid.
// https://github.com/ethereum/eth2.0-specs/blob/dev/specs/core/0_beacon-chain.md#attestations
func (bs *BeaconChainServer) AttestationPool(
	ctx context.Context, _ *ptypes.Empty,
) (*ethpb.AttestationPoolResponse, error) {

	atts, err := bs.pool.AttestationPoolNoVerify(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not fetch attestations: %v", err)
	}
	return &ethpb.AttestationPoolResponse{
		Attestations: atts,
	}, nil
}

// ListBlocks retrieves blocks by root, slot, or epoch.
//
// The server may return multiple blocks in the case that a slot or epoch is
// provided as the filter criteria. The server may return an empty list when
// no blocks in their database match the filter criteria. This RPC should
// not return NOT_FOUND. Only one filter criteria should be used.
func (bs *BeaconChainServer) ListBlocks(
	ctx context.Context, req *ethpb.ListBlocksRequest,
) (*ethpb.ListBlocksResponse, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	switch q := req.QueryFilter.(type) {
	case *ethpb.ListBlocksRequest_Epoch:
		startSlot := q.Epoch * params.BeaconConfig().SlotsPerEpoch
		endSlot := startSlot + params.BeaconConfig().SlotsPerEpoch - 1

		blks, err := bs.beaconDB.Blocks(ctx, filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get blocks: %v", err)
		}

		numBlks := len(blks)
		if numBlks == 0 {
			return &ethpb.ListBlocksResponse{Blocks: make([]*ethpb.BeaconBlock, 0), TotalSize: 0}, nil
		}

		start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numBlks)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not paginate blocks: %v", err)
		}

		return &ethpb.ListBlocksResponse{
			Blocks:        blks[start:end],
			TotalSize:     int32(numBlks),
			NextPageToken: nextPageToken,
		}, nil

	case *ethpb.ListBlocksRequest_Root:
		blk, err := bs.beaconDB.Block(ctx, bytesutil.ToBytes32(q.Root))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve block: %v", err)
		}

		if blk == nil {
			return &ethpb.ListBlocksResponse{Blocks: []*ethpb.BeaconBlock{}, TotalSize: 0}, nil
		}

		return &ethpb.ListBlocksResponse{
			Blocks:    []*ethpb.BeaconBlock{blk},
			TotalSize: 1,
		}, nil

	case *ethpb.ListBlocksRequest_Slot:
		blks, err := bs.beaconDB.Blocks(ctx, filters.NewFilter().SetStartSlot(q.Slot).SetEndSlot(q.Slot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve blocks for slot %d: %v", q.Slot, err)
		}

		numBlks := len(blks)
		if numBlks == 0 {
			return &ethpb.ListBlocksResponse{Blocks: []*ethpb.BeaconBlock{}, TotalSize: 0}, nil
		}

		start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numBlks)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not paginate blocks: %v", err)
		}

		return &ethpb.ListBlocksResponse{
			Blocks:        blks[start:end],
			TotalSize:     int32(numBlks),
			NextPageToken: nextPageToken,
		}, nil
	}

	return nil, status.Errorf(codes.InvalidArgument, "must satisfy one of the filter requirement")
}

// GetChainHead retrieves information about the head of the beacon chain from
// the view of the beacon chain node.
//
// This includes the head block slot and root as well as information about
// the most recent finalized and justified slots.
func (bs *BeaconChainServer) GetChainHead(ctx context.Context, _ *ptypes.Empty) (*ethpb.ChainHead, error) {
	finalizedCheckpoint := bs.headFetcher.HeadState().FinalizedCheckpoint
	justifiedCheckpoint := bs.headFetcher.HeadState().CurrentJustifiedCheckpoint
	prevJustifiedCheckpoint := bs.headFetcher.HeadState().PreviousJustifiedCheckpoint

	return &ethpb.ChainHead{
		BlockRoot:                  bs.headFetcher.HeadRoot(),
		BlockSlot:                  bs.headFetcher.HeadSlot(),
		FinalizedBlockRoot:         finalizedCheckpoint.Root,
		FinalizedSlot:              finalizedCheckpoint.Epoch * params.BeaconConfig().SlotsPerEpoch,
		JustifiedBlockRoot:         justifiedCheckpoint.Root,
		JustifiedSlot:              justifiedCheckpoint.Epoch * params.BeaconConfig().SlotsPerEpoch,
		PreviousJustifiedBlockRoot: prevJustifiedCheckpoint.Root,
		PreviousJustifiedSlot:      prevJustifiedCheckpoint.Epoch * params.BeaconConfig().SlotsPerEpoch,
	}, nil
}

// ListValidatorBalances retrieves the validator balances for a given set of public keys.
// An optional Epoch parameter is provided to request historical validator balances from
// archived, persistent data.
func (bs *BeaconChainServer) ListValidatorBalances(
	ctx context.Context,
	req *ethpb.GetValidatorBalancesRequest) (*ethpb.ValidatorBalances, error) {

	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	res := make([]*ethpb.ValidatorBalances_Balance, 0)
	filtered := map[uint64]bool{} // track filtered validators to prevent duplication in the response.

	headState := bs.headFetcher.HeadState()
	var requestingGenesis bool
	var epoch uint64
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorBalancesRequest_Epoch:
		epoch = q.Epoch
	case *ethpb.GetValidatorBalancesRequest_Genesis:
		requestingGenesis = q.Genesis
	default:
		epoch = helpers.CurrentEpoch(headState)
	}

	var balances []uint64
	var err error
	validators := headState.Validators
	if requestingGenesis {
		balances, err = bs.beaconDB.ArchivedBalances(ctx, 0 /* genesis epoch */)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "could not retrieve balances for epoch %d", epoch)
		}
	} else if !requestingGenesis && epoch < helpers.CurrentEpoch(headState) {
		balances, err = bs.beaconDB.ArchivedBalances(ctx, epoch)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "could not retrieve balances for epoch %d", epoch)
		}
	} else {
		balances = headState.Balances
	}

	for _, pubKey := range req.PublicKeys {
		// Skip empty public key
		if len(pubKey) == 0 {
			continue
		}

		index, ok, err := bs.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
		}
		if !ok {
			return nil, status.Errorf(codes.Internal, "could not find validator index for public key  %#x not found", pubKey)
		}

		filtered[index] = true

		if int(index) >= len(balances) {
			return nil, status.Errorf(codes.OutOfRange, "validator index %d >= balance list %d",
				index, len(balances))
		}

		res = append(res, &ethpb.ValidatorBalances_Balance{
			PublicKey: pubKey,
			Index:     index,
			Balance:   balances[index],
		})
	}

	for _, index := range req.Indices {
		if int(index) >= len(balances) {
			if epoch <= helpers.CurrentEpoch(headState) {
				return nil, status.Errorf(codes.OutOfRange, "validator index %d does not exist in historical balances",
					index)
			}
			return nil, status.Errorf(codes.OutOfRange, "validator index %d >= balance list %d",
				index, len(balances))
		}

		if !filtered[index] {
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: validators[index].PublicKey,
				Index:     index,
				Balance:   balances[index],
			})
		}
	}

	if len(req.Indices) == 0 && len(req.PublicKeys) == 0 {
		// return everything.
		for i := 0; i < len(headState.Balances); i++ {
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: headState.Validators[i].PublicKey,
				Index:     uint64(i),
				Balance:   balances[i],
			})
		}
	}

	balancesCount := len(res)
	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), balancesCount)
	if err != nil {
		return nil, err
	}
	return &ethpb.ValidatorBalances{
		Epoch:         epoch,
		Balances:      res[start:end],
		TotalSize:     int32(balancesCount),
		NextPageToken: nextPageToken,
	}, nil
}

// GetValidators retrieves the current list of active validators with an optional historical epoch flag to
// to retrieve validator set in time.
func (bs *BeaconChainServer) GetValidators(
	ctx context.Context,
	req *ethpb.GetValidatorsRequest,
) (*ethpb.Validators, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	headState := bs.headFetcher.HeadState()
	requestedEpoch := helpers.CurrentEpoch(headState)
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorsRequest_Genesis:
		if q.Genesis {
			requestedEpoch = 0
		}
	case *ethpb.GetValidatorsRequest_Epoch:
		requestedEpoch = q.Epoch
	}

	finalizedEpoch := bs.finalizationFetcher.FinalizedCheckpt().Epoch
	validators := headState.Validators
	if requestedEpoch < finalizedEpoch {
		stopIdx := len(validators)
		for idx, val := range validators {
			// The first time we see a validator with an activation epoch > the requested epoch,
			// we know this validator is from the future relative to what the request wants.
			if val.ActivationEpoch > requestedEpoch {
				stopIdx = idx
				break
			}
		}
		validators = validators[:stopIdx]
	}

	validatorCount := len(validators)
	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), validatorCount)
	if err != nil {
		return nil, err
	}

	return &ethpb.Validators{
		Validators:    validators[start:end],
		TotalSize:     int32(validatorCount),
		NextPageToken: nextPageToken,
	}, nil
}

// GetValidatorActiveSetChanges retrieves the active set changes for a given epoch.
//
// This data includes any activations, voluntary exits, and involuntary
// ejections.
func (bs *BeaconChainServer) GetValidatorActiveSetChanges(
	ctx context.Context, req *ethpb.GetValidatorActiveSetChangesRequest,
) (*ethpb.ActiveSetChanges, error) {
	headState := bs.headFetcher.HeadState()
	requestedEpoch := helpers.CurrentEpoch(headState)
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorActiveSetChangesRequest_Genesis:
		if q.Genesis {
			requestedEpoch = 0
		}
	case *ethpb.GetValidatorActiveSetChangesRequest_Epoch:
		requestedEpoch = q.Epoch
	}

	activatedIndices := make([]uint64, 0)
	slashedIndices := make([]uint64, 0)
	exitedIndices := make([]uint64, 0)
	finalizedEpoch := bs.finalizationFetcher.FinalizedCheckpt().Epoch
	var err error

	if requestedEpoch < finalizedEpoch {
		archivedChanges, err := bs.beaconDB.ArchivedActiveValidatorChanges(ctx, requestedEpoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch archived active validator changes: %v", err)
		}
		activatedIndices = archivedChanges.Activated
		slashedIndices = archivedChanges.Slashed
		exitedIndices = archivedChanges.Exited
	} else {
		activatedIndices = validators.ActivatedValidatorIndices(headState)
		slashedIndices = validators.SlashedValidatorIndices(headState)
		exitedIndices, err = validators.ExitedValidatorIndices(headState)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not determine exited validator indices: %v", err)
		}
	}

	// We retrieve the public keys for the indices.
	activatedKeys := make([][]byte, len(activatedIndices))
	slashedKeys := make([][]byte, len(slashedIndices))
	exitedKeys := make([][]byte, len(exitedIndices))
	for i, idx := range activatedIndices {
		activatedKeys[i] = headState.Validators[idx].PublicKey
	}
	for i, idx := range slashedIndices {
		slashedKeys[i] = headState.Validators[idx].PublicKey
	}
	for i, idx := range exitedIndices {
		exitedKeys[i] = headState.Validators[idx].PublicKey
	}
	return &ethpb.ActiveSetChanges{
		Epoch:               requestedEpoch,
		ActivatedPublicKeys: activatedKeys,
		ExitedPublicKeys:    exitedKeys,
		SlashedPublicKeys:   slashedKeys,
	}, nil
}

// GetValidatorQueue retrieves the current validator queue information.
func (bs *BeaconChainServer) GetValidatorQueue(
	ctx context.Context, _ *ptypes.Empty,
) (*ethpb.ValidatorQueue, error) {
	headState := bs.headFetcher.HeadState()
	// Queue the validators whose eligible to activate and sort them by activation eligibility epoch number.
	// Additionally, determine those validators queued to exit
	awaitingExit := make([]uint64, 0)
	exitEpochs := make([]uint64, 0)
	activationQ := make([]uint64, 0)
	for idx, validator := range headState.Validators {
		eligibleActivated := validator.ActivationEligibilityEpoch != params.BeaconConfig().FarFutureEpoch
		canBeActive := validator.ActivationEpoch >= helpers.DelayedActivationExitEpoch(headState.FinalizedCheckpoint.Epoch)
		if eligibleActivated && canBeActive {
			activationQ = append(activationQ, uint64(idx))
		}
		if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochs = append(exitEpochs, validator.ExitEpoch)
			awaitingExit = append(awaitingExit, uint64(idx))
		}
	}
	sort.Slice(activationQ, func(i, j int) bool {
		return headState.Validators[i].ActivationEligibilityEpoch < headState.Validators[j].ActivationEligibilityEpoch
	})
	sort.Slice(awaitingExit, func(i, j int) bool {
		return headState.Validators[i].WithdrawableEpoch < headState.Validators[j].WithdrawableEpoch
	})

	// Only activate just enough validators according to the activation churn limit.
	activationQueueChurn := len(activationQ)
	churnLimit, err := helpers.ValidatorChurnLimit(headState)
	if err != nil {
		return nil, errors.Wrap(err, "could not get churn limit")
	}

	exitQueueEpoch := uint64(0)
	for _, i := range exitEpochs {
		if exitQueueEpoch < i {
			exitQueueEpoch = i
		}
	}
	exitQueueChurn := 0
	for _, val := range headState.Validators {
		if val.ExitEpoch == exitQueueEpoch {
			exitQueueChurn++
		}
	}
	// Prevent churn limit from causing index out of bound issues.
	if int(churnLimit) < activationQueueChurn {
		activationQueueChurn = int(churnLimit)
	}
	if int(churnLimit) < exitQueueChurn {
		// If we are above the churn limit, we simply increase the churn by one.
		exitQueueEpoch++
		exitQueueChurn = int(churnLimit)
	}

	// We use the exit queue churn to determine if we have passed a churn limit.
	minEpoch := exitQueueEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	exitQueueIndices := make([]uint64, 0)
	for _, valIdx := range awaitingExit {
		if headState.Validators[valIdx].WithdrawableEpoch < minEpoch {
			exitQueueIndices = append(exitQueueIndices, valIdx)
		}
	}

	// Get the public keys for the validators in the queues up to the allowed churn limits.
	activationQueueKeys := make([][]byte, len(activationQ))
	exitQueueKeys := make([][]byte, len(exitQueueIndices))
	for i, idx := range activationQ {
		activationQueueKeys[i] = headState.Validators[idx].PublicKey
	}
	for i, idx := range exitQueueIndices {
		exitQueueKeys[i] = headState.Validators[idx].PublicKey
	}

	return &ethpb.ValidatorQueue{
		ChurnLimit:           churnLimit,
		ActivationPublicKeys: activationQueueKeys,
		ExitPublicKeys:       exitQueueKeys,
	}, nil
}

// ListValidatorAssignments retrieves the validator assignments for a given epoch,
// optional validator indices or public keys may be included to filter validator assignments.
func (bs *BeaconChainServer) ListValidatorAssignments(
	ctx context.Context, req *ethpb.ListValidatorAssignmentsRequest,
) (*ethpb.ValidatorAssignments, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	var res []*ethpb.ValidatorAssignments_CommitteeAssignment
	headState := bs.headFetcher.HeadState()
	filtered := map[uint64]bool{} // track filtered validators to prevent duplication in the response.
	filteredIndices := make([]uint64, 0)
	requestedEpoch := helpers.CurrentEpoch(headState)

	switch q := req.QueryFilter.(type) {
	case *ethpb.ListValidatorAssignmentsRequest_Genesis:
		if q.Genesis {
			requestedEpoch = 0
		}
	case *ethpb.ListValidatorAssignmentsRequest_Epoch:
		requestedEpoch = q.Epoch
	}

	// Filter out assignments by public keys.
	for _, pubKey := range req.PublicKeys {
		index, ok, err := bs.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
		}
		if !ok {
			return nil, status.Errorf(codes.NotFound, "could not find validator index for public key  %#x not found", pubKey)
		}
		filtered[index] = true
		filteredIndices = append(filteredIndices, index)
	}

	// Filter out assignments by validator indices.
	for _, index := range req.Indices {
		if !filtered[index] {
			filteredIndices = append(filteredIndices, index)
		}
	}

	activeIndices, err := helpers.ActiveValidatorIndices(headState, requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve active validator indices: %v", err)
	}
	if len(filteredIndices) == 0 {
		// If no filter was specified, return assignments from active validator indices with pagination.
		filteredIndices = activeIndices
	}

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), len(filteredIndices))
	if err != nil {
		return nil, err
	}

	shouldFetchFromArchive := requestedEpoch < bs.finalizationFetcher.FinalizedCheckpt().Epoch

	for _, index := range filteredIndices[start:end] {
		if int(index) >= len(headState.Validators) {
			return nil, status.Errorf(codes.InvalidArgument, "validator index %d >= validator count %d",
				index, len(headState.Validators))
		}
		if shouldFetchFromArchive {
			archivedInfo, err := bs.beaconDB.ArchivedCommitteeInfo(ctx, requestedEpoch)
			if err != nil {
				return nil, status.Errorf(
					codes.Internal,
					"could not retrieve archived committee info for epoch %d",
					requestedEpoch,
				)
			}
			if archivedInfo == nil {
				return nil, status.Errorf(
					codes.NotFound,
					"no archival committee info found for epoch %d",
					requestedEpoch,
				)
			}
			archivedBalances, err := bs.beaconDB.ArchivedBalances(ctx, requestedEpoch)
			if err != nil {
				return nil, status.Errorf(
					codes.Internal,
					"could not retrieve archived balances for epoch %d",
					requestedEpoch,
				)
			}
			if archivedBalances == nil {
				return nil, status.Errorf(
					codes.NotFound,
					"no archival balances found for epoch %d",
					requestedEpoch,
				)
			}
			committee, committeeIndex, attesterSlot, proposerSlot, err := archivedValidatorCommittee(
				requestedEpoch,
				index,
				archivedInfo,
				activeIndices,
				archivedBalances,
			)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "could not retrieve archived assignment for validator %d: %v", index, err)
			}
			assign := &ethpb.ValidatorAssignments_CommitteeAssignment{
				BeaconCommittees: committee,
				CommitteeIndex:   committeeIndex,
				AttesterSlot:     attesterSlot,
				ProposerSlot:     proposerSlot,
				PublicKey:        headState.Validators[index].PublicKey,
			}
			res = append(res, assign)
			continue
		}
		committee, committeeIndex, attesterSlot, proposerSlot, err := helpers.CommitteeAssignment(headState, requestedEpoch, index)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve assignment for validator %d: %v", index, err)
		}
		assign := &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: committee,
			CommitteeIndex:   committeeIndex,
			AttesterSlot:     attesterSlot,
			ProposerSlot:     proposerSlot,
			PublicKey:        headState.Validators[index].PublicKey,
		}
		res = append(res, assign)
	}

	return &ethpb.ValidatorAssignments{
		Epoch:         requestedEpoch,
		Assignments:   res,
		NextPageToken: nextPageToken,
		TotalSize:     int32(len(filteredIndices)),
	}, nil
}

// Computes validator assignments for an epoch and validator index using archived committee
// information, archived balances, and a set of active validators.
func archivedValidatorCommittee(
	epoch uint64,
	validatorIndex uint64,
	archivedInfo *ethpb.ArchivedCommitteeInfo,
	activeIndices []uint64,
	archivedBalances []uint64,
) ([]uint64, uint64, uint64, uint64, error) {
	committeeCount := archivedInfo.CommitteeCount
	proposerSeed := bytesutil.ToBytes32(archivedInfo.ProposerSeed)
	attesterSeed := bytesutil.ToBytes32(archivedInfo.AttesterSeed)

	startSlot := helpers.StartSlot(epoch)
	proposerIndexToSlot := make(map[uint64]uint64)
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		seedWithSlot := append(proposerSeed[:], bytesutil.Bytes8(slot)...)
		seedWithSlotHash := hashutil.Hash(seedWithSlot)
		i, err := archivedProposerIndex(activeIndices, archivedBalances, seedWithSlotHash)
		if err != nil {
			return nil, 0, 0, 0, errors.Wrapf(err, "could not check proposer at slot %d", slot)
		}
		proposerIndexToSlot[i] = slot
	}
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		var countAtSlot = uint64(len(activeIndices)) / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize
		if countAtSlot > params.BeaconConfig().MaxCommitteesPerSlot {
			countAtSlot = params.BeaconConfig().MaxCommitteesPerSlot
		}
		if countAtSlot == 0 {
			countAtSlot = 1
		}
		for i := uint64(0); i < countAtSlot; i++ {
			epochOffset := i + (slot%params.BeaconConfig().SlotsPerEpoch)*countAtSlot
			committee, err := helpers.ComputeCommittee(activeIndices, attesterSeed, epochOffset, committeeCount)
			if err != nil {
				return nil, 0, 0, 0, errors.Wrap(err, "could not compute committee")
			}
			for _, index := range committee {
				if validatorIndex == index {
					proposerSlot, _ := proposerIndexToSlot[validatorIndex]
					return committee, i, slot, proposerSlot, nil
				}
			}
		}
	}
	return nil, 0, 0, 0, fmt.Errorf("could not find committee for validator index %d", validatorIndex)
}

func archivedProposerIndex(activeIndices []uint64, activeBalances []uint64, seed [32]byte) (uint64, error) {
	length := uint64(len(activeIndices))
	if length == 0 {
		return 0, errors.New("empty indices list")
	}
	maxRandomByte := uint64(1<<8 - 1)
	for i := uint64(0); ; i++ {
		candidateIndex, err := helpers.ComputeShuffledIndex(i%length, length, seed, true)
		if err != nil {
			return 0, err
		}
		b := append(seed[:], bytesutil.Bytes8(i/32)...)
		randomByte := hashutil.Hash(b)[i%32]
		effectiveBalance := activeBalances[candidateIndex]
		if effectiveBalance >= params.BeaconConfig().MaxEffectiveBalance {
			// if the actual balance is greater than or equal to the max effective balance,
			// we just determine the proposer index using config.MaxEffectiveBalance.
			effectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		}
		if effectiveBalance*maxRandomByte >= params.BeaconConfig().MaxEffectiveBalance*uint64(randomByte) {
			return candidateIndex, nil
		}
	}
}

// GetValidatorParticipation retrieves the validator participation information for a given epoch,
// it returns the information about validator's participation rate in voting on the proof of stake
// rules based on their balance compared to the total active validator balance.
func (bs *BeaconChainServer) GetValidatorParticipation(
	ctx context.Context, req *ethpb.GetValidatorParticipationRequest,
) (*ethpb.ValidatorParticipationResponse, error) {
	headState := bs.headFetcher.HeadState()
	currentEpoch := helpers.SlotToEpoch(headState.Slot)

	var requestedEpoch uint64
	var isGenesis bool
	switch q := req.QueryFilter.(type) {
	case *ethpb.GetValidatorParticipationRequest_Genesis:
		isGenesis = q.Genesis
	case *ethpb.GetValidatorParticipationRequest_Epoch:
		requestedEpoch = q.Epoch
	default:
		requestedEpoch = currentEpoch
	}

	if requestedEpoch > helpers.SlotToEpoch(headState.Slot) {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"cannot request data from an epoch in the future: req.Epoch %d, currentEpoch %d", requestedEpoch, currentEpoch,
		)
	}
	// If the request is from genesis or another past epoch, we look into our archived
	// data to find it and return it if it exists.
	if isGenesis {
		participation, err := bs.beaconDB.ArchivedValidatorParticipation(ctx, 0)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch archived participation: %v", err)
		}
		if participation == nil {
			return nil, status.Error(codes.NotFound, "could not find archival data for epoch 0")
		}
		return &ethpb.ValidatorParticipationResponse{
			Epoch:         0,
			Finalized:     true,
			Participation: participation,
		}, nil
	} else if requestedEpoch < helpers.SlotToEpoch(headState.Slot) {
		participation, err := bs.beaconDB.ArchivedValidatorParticipation(ctx, requestedEpoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch archived participation: %v", err)
		}
		if participation == nil {
			return nil, status.Errorf(codes.NotFound, "could not find archival data for epoch %d", requestedEpoch)
		}
		finalizedEpoch := bs.finalizationFetcher.FinalizedCheckpt().Epoch
		// If the epoch we requested is <= the finalized epoch, we consider it finalized as well.
		finalized := requestedEpoch <= finalizedEpoch
		return &ethpb.ValidatorParticipationResponse{
			Epoch:         requestedEpoch,
			Finalized:     finalized,
			Participation: participation,
		}, nil
	}
	// Else if the request is for the current epoch, we compute validator participation
	// right away and return the result based on the head state.
	participation, err := epoch.ComputeValidatorParticipation(headState)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not compute participation: %v", err)
	}
	return &ethpb.ValidatorParticipationResponse{
		Epoch:         currentEpoch,
		Finalized:     false, // The current epoch can never be finalized.
		Participation: participation,
	}, nil
}
