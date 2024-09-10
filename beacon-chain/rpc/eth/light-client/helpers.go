package lightclient

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/pkg/errors"

	lightclient "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/light-client"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/proto/migration"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func createLightClientBootstrap(ctx context.Context, state state.BeaconState, blk interfaces.ReadOnlyBeaconBlock) (*structs.LightClientBootstrap, error) {
	switch blk.Version() {
	case version.Phase0:
		return nil, fmt.Errorf("light client bootstrap is not supported for phase0")
	case version.Altair, version.Bellatrix:
		return createLightClientBootstrapAltair(ctx, state)
	case version.Capella:
		return createLightClientBootstrapCapella(ctx, state, blk)
	case version.Deneb, version.Electra:
		return createLightClientBootstrapDeneb(ctx, state, blk)
	}
	return nil, fmt.Errorf("unsupported block version %s", version.String(blk.Version()))
}

// createLightClientBootstrapAltair - implements https://github.com/ethereum/consensus-specs/blob/3d235740e5f1e641d3b160c8688f26e7dc5a1894/specs/altair/light-client/full-node.md#create_light_client_bootstrap
// def create_light_client_bootstrap(state: BeaconState) -> LightClientBootstrap:
//
//	assert compute_epoch_at_slot(state.slot) >= ALTAIR_FORK_EPOCH
//	assert state.slot == state.latest_block_header.slot
//
//	return LightClientBootstrap(
//	    header=BeaconBlockHeader(
//	        slot=state.latest_block_header.slot,
//	        proposer_index=state.latest_block_header.proposer_index,
//	        parent_root=state.latest_block_header.parent_root,
//	        state_root=hash_tree_root(state),
//	        body_root=state.latest_block_header.body_root,
//	    ),
//	    current_sync_committee=state.current_sync_committee,
//	    current_sync_committee_branch=compute_merkle_proof_for_state(state, CURRENT_SYNC_COMMITTEE_INDEX)
//	)
func createLightClientBootstrapAltair(ctx context.Context, state state.BeaconState) (*structs.LightClientBootstrap, error) {
	// assert compute_epoch_at_slot(state.slot) >= ALTAIR_FORK_EPOCH
	if slots.ToEpoch(state.Slot()) < params.BeaconConfig().AltairForkEpoch {
		return nil, fmt.Errorf("light client bootstrap is not supported before Altair, invalid slot %d", state.Slot())
	}

	// assert state.slot == state.latest_block_header.slot
	latestBlockHeader := state.LatestBlockHeader()
	if state.Slot() != latestBlockHeader.Slot {
		return nil, fmt.Errorf("state slot %d not equal to latest block header slot %d", state.Slot(), latestBlockHeader.Slot)
	}

	// Prepare data
	currentSyncCommittee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}

	committee := structs.SyncCommitteeFromConsensus(currentSyncCommittee)

	currentSyncCommitteeProof, err := state.CurrentSyncCommitteeProof(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee proof")
	}

	branch := make([]string, fieldparams.NextSyncCommitteeBranchDepth)
	for i, proof := range currentSyncCommitteeProof {
		branch[i] = hexutil.Encode(proof)
	}

	beacon := structs.BeaconBlockHeaderFromConsensus(latestBlockHeader)
	if beacon == nil {
		return nil, fmt.Errorf("could not get beacon block header")
	}
	header := &structs.LightClientHeader{
		Beacon: beacon,
	}

	// Above shared util function won't calculate state root, so we need to do it manually
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state root")
	}
	header.Beacon.StateRoot = hexutil.Encode(stateRoot[:])

	headerJson, err := json.Marshal(header)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert header to raw message")
	}

	// Return result
	result := &structs.LightClientBootstrap{
		Header:                     headerJson,
		CurrentSyncCommittee:       committee,
		CurrentSyncCommitteeBranch: branch,
	}

	return result, nil
}

func createLightClientBootstrapCapella(ctx context.Context, state state.BeaconState, block interfaces.ReadOnlyBeaconBlock) (*structs.LightClientBootstrap, error) {
	// assert compute_epoch_at_slot(state.slot) >= CAPELLA_FORK_EPOCH
	if slots.ToEpoch(state.Slot()) < params.BeaconConfig().CapellaForkEpoch {
		return nil, fmt.Errorf("creating Capella light client bootstrap is not supported before Capella, invalid slot %d", state.Slot())
	}

	// assert state.slot == state.latest_block_header.slot
	latestBlockHeader := state.LatestBlockHeader()
	if state.Slot() != latestBlockHeader.Slot {
		return nil, fmt.Errorf("state slot %d not equal to latest block header slot %d", state.Slot(), latestBlockHeader.Slot)
	}

	// Prepare data
	currentSyncCommittee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}

	committee := structs.SyncCommitteeFromConsensus(currentSyncCommittee)

	currentSyncCommitteeProof, err := state.CurrentSyncCommitteeProof(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee proof")
	}

	branch := make([]string, fieldparams.NextSyncCommitteeBranchDepth)
	for i, proof := range currentSyncCommitteeProof {
		branch[i] = hexutil.Encode(proof)
	}

	beacon := structs.BeaconBlockHeaderFromConsensus(latestBlockHeader)

	payloadInterface, err := block.Body().Execution()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload")
	}
	transactionsRoot, err := payloadInterface.TransactionsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		transactions, err := payloadInterface.Transactions()
		if err != nil {
			return nil, errors.Wrap(err, "could not get transactions")
		}
		transactionsRootArray, err := ssz.TransactionsRoot(transactions)
		if err != nil {
			return nil, errors.Wrap(err, "could not get transactions root")
		}
		transactionsRoot = transactionsRootArray[:]
	} else if err != nil {
		return nil, errors.Wrap(err, "could not get transactions root")
	}
	withdrawalsRoot, err := payloadInterface.WithdrawalsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		withdrawals, err := payloadInterface.Withdrawals()
		if err != nil {
			return nil, errors.Wrap(err, "could not get withdrawals")
		}
		withdrawalsRootArray, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		if err != nil {
			return nil, errors.Wrap(err, "could not get withdrawals root")
		}
		withdrawalsRoot = withdrawalsRootArray[:]
	}
	executionPayloadHeader := &structs.ExecutionPayloadHeaderCapella{
		ParentHash:       hexutil.Encode(payloadInterface.ParentHash()),
		FeeRecipient:     hexutil.Encode(payloadInterface.FeeRecipient()),
		StateRoot:        hexutil.Encode(payloadInterface.StateRoot()),
		ReceiptsRoot:     hexutil.Encode(payloadInterface.ReceiptsRoot()),
		LogsBloom:        hexutil.Encode(payloadInterface.LogsBloom()),
		PrevRandao:       hexutil.Encode(payloadInterface.PrevRandao()),
		BlockNumber:      hexutil.EncodeUint64(payloadInterface.BlockNumber()),
		GasLimit:         hexutil.EncodeUint64(payloadInterface.GasLimit()),
		GasUsed:          hexutil.EncodeUint64(payloadInterface.GasUsed()),
		Timestamp:        hexutil.EncodeUint64(payloadInterface.Timestamp()),
		ExtraData:        hexutil.Encode(payloadInterface.ExtraData()),
		BaseFeePerGas:    hexutil.Encode(payloadInterface.BaseFeePerGas()),
		BlockHash:        hexutil.Encode(payloadInterface.BlockHash()),
		TransactionsRoot: hexutil.Encode(transactionsRoot),
		WithdrawalsRoot:  hexutil.Encode(withdrawalsRoot),
	}

	executionPayloadProof, err := blocks.PayloadProof(ctx, block)
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload proof")
	}
	executionPayloadProofStr := make([]string, len(executionPayloadProof))
	for i, proof := range executionPayloadProof {
		executionPayloadProofStr[i] = hexutil.Encode(proof)
	}
	header := &structs.LightClientHeaderCapella{
		Beacon:          beacon,
		Execution:       executionPayloadHeader,
		ExecutionBranch: executionPayloadProofStr,
	}

	// Above shared util function won't calculate state root, so we need to do it manually
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state root")
	}
	header.Beacon.StateRoot = hexutil.Encode(stateRoot[:])

	headerJson, err := json.Marshal(header)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert header to raw message")
	}

	// Return result
	result := &structs.LightClientBootstrap{
		Header:                     headerJson,
		CurrentSyncCommittee:       committee,
		CurrentSyncCommitteeBranch: branch,
	}

	return result, nil
}

func createLightClientBootstrapDeneb(ctx context.Context, state state.BeaconState, block interfaces.ReadOnlyBeaconBlock) (*structs.LightClientBootstrap, error) {
	// assert compute_epoch_at_slot(state.slot) >= DENEB_FORK_EPOCH
	if slots.ToEpoch(state.Slot()) < params.BeaconConfig().DenebForkEpoch {
		return nil, fmt.Errorf("creating Deneb light client bootstrap is not supported before Deneb, invalid slot %d", state.Slot())
	}

	// assert state.slot == state.latest_block_header.slot
	latestBlockHeader := state.LatestBlockHeader()
	if state.Slot() != latestBlockHeader.Slot {
		return nil, fmt.Errorf("state slot %d not equal to latest block header slot %d", state.Slot(), latestBlockHeader.Slot)
	}

	// Prepare data
	currentSyncCommittee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}

	committee := structs.SyncCommitteeFromConsensus(currentSyncCommittee)

	currentSyncCommitteeProof, err := state.CurrentSyncCommitteeProof(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee proof")
	}

	branch := make([]string, fieldparams.NextSyncCommitteeBranchDepth)
	for i, proof := range currentSyncCommitteeProof {
		branch[i] = hexutil.Encode(proof)
	}

	beacon := structs.BeaconBlockHeaderFromConsensus(latestBlockHeader)

	payloadInterface, err := block.Body().Execution()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload")
	}
	transactionsRoot, err := payloadInterface.TransactionsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		transactions, err := payloadInterface.Transactions()
		if err != nil {
			return nil, errors.Wrap(err, "could not get transactions")
		}
		transactionsRootArray, err := ssz.TransactionsRoot(transactions)
		if err != nil {
			return nil, errors.Wrap(err, "could not get transactions root")
		}
		transactionsRoot = transactionsRootArray[:]
	} else if err != nil {
		return nil, errors.Wrap(err, "could not get transactions root")
	}
	withdrawalsRoot, err := payloadInterface.WithdrawalsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		withdrawals, err := payloadInterface.Withdrawals()
		if err != nil {
			return nil, errors.Wrap(err, "could not get withdrawals")
		}
		withdrawalsRootArray, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		if err != nil {
			return nil, errors.Wrap(err, "could not get withdrawals root")
		}
		withdrawalsRoot = withdrawalsRootArray[:]
	}
	executionPayloadHeader := &structs.ExecutionPayloadHeaderDeneb{
		ParentHash:       hexutil.Encode(payloadInterface.ParentHash()),
		FeeRecipient:     hexutil.Encode(payloadInterface.FeeRecipient()),
		StateRoot:        hexutil.Encode(payloadInterface.StateRoot()),
		ReceiptsRoot:     hexutil.Encode(payloadInterface.ReceiptsRoot()),
		LogsBloom:        hexutil.Encode(payloadInterface.LogsBloom()),
		PrevRandao:       hexutil.Encode(payloadInterface.PrevRandao()),
		BlockNumber:      hexutil.EncodeUint64(payloadInterface.BlockNumber()),
		GasLimit:         hexutil.EncodeUint64(payloadInterface.GasLimit()),
		GasUsed:          hexutil.EncodeUint64(payloadInterface.GasUsed()),
		Timestamp:        hexutil.EncodeUint64(payloadInterface.Timestamp()),
		ExtraData:        hexutil.Encode(payloadInterface.ExtraData()),
		BaseFeePerGas:    hexutil.Encode(payloadInterface.BaseFeePerGas()),
		BlockHash:        hexutil.Encode(payloadInterface.BlockHash()),
		TransactionsRoot: hexutil.Encode(transactionsRoot),
		WithdrawalsRoot:  hexutil.Encode(withdrawalsRoot),
	}

	executionPayloadProof, err := blocks.PayloadProof(ctx, block)
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload proof")
	}
	executionPayloadProofStr := make([]string, len(executionPayloadProof))
	for i, proof := range executionPayloadProof {
		executionPayloadProofStr[i] = hexutil.Encode(proof)
	}
	header := &structs.LightClientHeaderDeneb{
		Beacon:          beacon,
		Execution:       executionPayloadHeader,
		ExecutionBranch: executionPayloadProofStr,
	}

	// Above shared util function won't calculate state root, so we need to do it manually
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state root")
	}
	header.Beacon.StateRoot = hexutil.Encode(stateRoot[:])

	headerJson, err := json.Marshal(header)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert header to raw message")
	}
	// Return result
	result := &structs.LightClientBootstrap{
		Header:                     headerJson,
		CurrentSyncCommittee:       committee,
		CurrentSyncCommitteeBranch: branch,
	}

	return result, nil
}

// createLightClientUpdate - implements https://github.
// com/ethereum/consensus-specs/blob/d70dcd9926a4bbe987f1b4e65c3e05bd029fcfb8/specs/altair/light-client/full-node.md#create_light_client_update
// def create_light_client_update(state: BeaconState,
//
//	                           block: SignedBeaconBlock,
//	                           attested_state: BeaconState,
//	                           finalized_block: Optional[SignedBeaconBlock]) -> LightClientUpdate:
//	assert compute_epoch_at_slot(attested_state.slot) >= ALTAIR_FORK_EPOCH
//	assert sum(block.message.body.sync_aggregate.sync_committee_bits) >= MIN_SYNC_COMMITTEE_PARTICIPANTS
//
//	assert state.slot == state.latest_block_header.slot
//	header = state.latest_block_header.copy()
//	header.state_root = hash_tree_root(state)
//	assert hash_tree_root(header) == hash_tree_root(block.message)
//	update_signature_period = compute_sync_committee_period(compute_epoch_at_slot(block.message.slot))
//
//	assert attested_state.slot == attested_state.latest_block_header.slot
//	attested_header = attested_state.latest_block_header.copy()
//	attested_header.state_root = hash_tree_root(attested_state)
//	assert hash_tree_root(attested_header) == block.message.parent_root
//	update_attested_period = compute_sync_committee_period(compute_epoch_at_slot(attested_header.slot))
//
//	# `next_sync_committee` is only useful if the message is signed by the current sync committee
//	if update_attested_period == update_signature_period:
//	    next_sync_committee = attested_state.next_sync_committee
//	    next_sync_committee_branch = compute_merkle_proof_for_state(attested_state, NEXT_SYNC_COMMITTEE_INDEX)
//	else:
//	    next_sync_committee = SyncCommittee()
//	    next_sync_committee_branch = [Bytes32() for _ in range(floorlog2(NEXT_SYNC_COMMITTEE_INDEX))]
//
//	# Indicate finality whenever possible
//	if finalized_block is not None:
//	    if finalized_block.message.slot != GENESIS_SLOT:
//	        finalized_header = BeaconBlockHeader(
//	            slot=finalized_block.message.slot,
//	            proposer_index=finalized_block.message.proposer_index,
//	            parent_root=finalized_block.message.parent_root,
//	            state_root=finalized_block.message.state_root,
//	            body_root=hash_tree_root(finalized_block.message.body),
//	        )
//	        assert hash_tree_root(finalized_header) == attested_state.finalized_checkpoint.root
//	    else:
//	        assert attested_state.finalized_checkpoint.root == Bytes32()
//	        finalized_header = BeaconBlockHeader()
//	    finality_branch = compute_merkle_proof_for_state(attested_state, FINALIZED_ROOT_INDEX)
//	else:
//	    finalized_header = BeaconBlockHeader()
//	    finality_branch = [Bytes32() for _ in range(floorlog2(FINALIZED_ROOT_INDEX))]
//
//	return LightClientUpdate(
//	    attested_header=attested_header,
//	    next_sync_committee=next_sync_committee,
//	    next_sync_committee_branch=next_sync_committee_branch,
//	    finalized_header=finalized_header,
//	    finality_branch=finality_branch,
//	    sync_aggregate=block.message.body.sync_aggregate,
//	    signature_slot=block.message.slot,
//	)
func createLightClientUpdate(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock) (*structs.LightClientUpdate, error) {
	result, err := lightclient.NewLightClientFinalityUpdateFromBeaconState(ctx, state, block, attestedState, finalizedBlock)
	if err != nil {
		return nil, err
	}

	// Generate next sync committee and proof
	var nextSyncCommittee *v2.SyncCommittee
	var nextSyncCommitteeBranch [][]byte

	// update_signature_period = compute_sync_committee_period(compute_epoch_at_slot(block.message.slot))
	updateSignaturePeriod := slots.SyncCommitteePeriod(slots.ToEpoch(block.Block().Slot()))

	// update_attested_period = compute_sync_committee_period(compute_epoch_at_slot(attested_header.slot))
	resultAttestedHeaderBeacon, err := result.AttestedHeader.GetBeacon()
	if err != nil {
		return nil, errors.Wrap(err, "could not get attested header beacon")
	}
	updateAttestedPeriod := slots.SyncCommitteePeriod(slots.ToEpoch(resultAttestedHeaderBeacon.Slot))

	if updateAttestedPeriod == updateSignaturePeriod {
		tempNextSyncCommittee, err := attestedState.NextSyncCommittee()
		if err != nil {
			return nil, errors.Wrap(err, "could not get next sync committee")
		}

		nextSyncCommittee = &v2.SyncCommittee{
			Pubkeys:         tempNextSyncCommittee.Pubkeys,
			AggregatePubkey: tempNextSyncCommittee.AggregatePubkey,
		}

		nextSyncCommitteeBranch, err = attestedState.NextSyncCommitteeProof(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get next sync committee proof")
		}
	} else {
		syncCommitteeSize := params.BeaconConfig().SyncCommitteeSize
		pubKeys := make([][]byte, syncCommitteeSize)
		for i := uint64(0); i < syncCommitteeSize; i++ {
			pubKeys[i] = make([]byte, fieldparams.BLSPubkeyLength)
		}
		nextSyncCommittee = &v2.SyncCommittee{
			Pubkeys:         pubKeys,
			AggregatePubkey: make([]byte, fieldparams.BLSPubkeyLength),
		}

		nextSyncCommitteeBranch = make([][]byte, fieldparams.NextSyncCommitteeBranchDepth)
		for i := 0; i < fieldparams.NextSyncCommitteeBranchDepth; i++ {
			nextSyncCommitteeBranch[i] = make([]byte, fieldparams.RootLength)
		}
	}

	result.NextSyncCommittee = nextSyncCommittee
	result.NextSyncCommitteeBranch = nextSyncCommitteeBranch
	res, err := newLightClientUpdateToJSON(result)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert light client update to JSON")
	}
	return res, nil
}

func newLightClientFinalityUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock) (*structs.LightClientUpdate, error) {
	result, err := lightclient.NewLightClientFinalityUpdateFromBeaconState(ctx, state, block, attestedState, finalizedBlock)
	if err != nil {
		return nil, err
	}

	res, err := newLightClientUpdateToJSON(result)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert light client update to JSON")
	}
	return res, nil
}

func newLightClientOptimisticUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState) (*structs.LightClientUpdate, error) {
	result, err := lightclient.NewLightClientOptimisticUpdateFromBeaconState(ctx, state, block, attestedState)
	if err != nil {
		return nil, err
	}

	res, err := newLightClientUpdateToJSON(result)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert light client update to JSON")
	}
	return res, nil
}

func branchToJSON(branchBytes [][]byte) []string {
	if branchBytes == nil {
		return nil
	}
	branch := make([]string, len(branchBytes))
	for i, root := range branchBytes {
		branch[i] = hexutil.Encode(root)
	}
	return branch
}

func syncAggregateToJSON(input *v1.SyncAggregate) *structs.SyncAggregate {
	if input == nil {
		return nil
	}
	return &structs.SyncAggregate{
		SyncCommitteeBits:      hexutil.Encode(input.SyncCommitteeBits),
		SyncCommitteeSignature: hexutil.Encode(input.SyncCommitteeSignature),
	}
}

func newLightClientUpdateToJSON(input *v2.LightClientUpdate) (*structs.LightClientUpdate, error) {
	if input == nil {
		return nil, errors.New("input is nil")
	}

	var nextSyncCommittee *structs.SyncCommittee
	if input.NextSyncCommittee != nil {
		nextSyncCommittee = structs.SyncCommitteeFromConsensus(migration.V2SyncCommitteeToV1Alpha1(input.NextSyncCommittee))
	}

	var finalizedHeader *structs.BeaconBlockHeader
	if input.FinalizedHeader != nil {
		inputFinalizedHeaderBeacon, err := input.FinalizedHeader.GetBeacon()
		if err != nil {
			return nil, errors.Wrap(err, "could not get finalized header beacon")
		}
		finalizedHeader = structs.BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(inputFinalizedHeaderBeacon))
	}

	inputAttestedHeaderBeacon, err := input.AttestedHeader.GetBeacon()
	if err != nil {
		return nil, errors.Wrap(err, "could not get attested header beacon")
	}
	attestedHeaderJson, err := json.Marshal(&structs.LightClientHeader{Beacon: structs.BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(inputAttestedHeaderBeacon))})
	if err != nil {
		return nil, errors.Wrap(err, "could not convert attested header to raw message")
	}
	finalizedHeaderJson, err := json.Marshal(&structs.LightClientHeader{Beacon: finalizedHeader})
	if err != nil {
		return nil, errors.Wrap(err, "could not convert finalized header to raw message")
	}
	result := &structs.LightClientUpdate{
		AttestedHeader:          attestedHeaderJson,
		NextSyncCommittee:       nextSyncCommittee,
		NextSyncCommitteeBranch: branchToJSON(input.NextSyncCommitteeBranch),
		FinalizedHeader:         finalizedHeaderJson,
		FinalityBranch:          branchToJSON(input.FinalityBranch),
		SyncAggregate:           syncAggregateToJSON(input.SyncAggregate),
		SignatureSlot:           strconv.FormatUint(uint64(input.SignatureSlot), 10),
	}

	return result, nil
}

func IsSyncCommitteeUpdate(update *v2.LightClientUpdate) bool {
	nextSyncCommitteeBranch := make([][]byte, fieldparams.NextSyncCommitteeBranchDepth)
	return !reflect.DeepEqual(update.NextSyncCommitteeBranch, nextSyncCommitteeBranch)
}

func IsFinalityUpdate(update *v2.LightClientUpdate) bool {
	finalityBranch := make([][]byte, lightclient.FinalityBranchNumOfLeaves)
	return !reflect.DeepEqual(update.FinalityBranch, finalityBranch)
}

func IsBetterUpdate(newUpdate, oldUpdate *v2.LightClientUpdate) (bool, error) {
	maxActiveParticipants := newUpdate.SyncAggregate.SyncCommitteeBits.Len()
	newNumActiveParticipants := newUpdate.SyncAggregate.SyncCommitteeBits.Count()
	oldNumActiveParticipants := oldUpdate.SyncAggregate.SyncCommitteeBits.Count()
	newHasSupermajority := newNumActiveParticipants*3 >= maxActiveParticipants*2
	oldHasSupermajority := oldNumActiveParticipants*3 >= maxActiveParticipants*2

	if newHasSupermajority != oldHasSupermajority {
		return newHasSupermajority, nil
	}
	if !newHasSupermajority && newNumActiveParticipants != oldNumActiveParticipants {
		return newNumActiveParticipants > oldNumActiveParticipants, nil
	}

	newUpdateAttestedHeaderBeacon, err := newUpdate.AttestedHeader.GetBeacon()
	if err != nil {
		return false, errors.Wrap(err, "could not get attested header beacon")
	}
	oldUpdateAttestedHeaderBeacon, err := oldUpdate.AttestedHeader.GetBeacon()
	if err != nil {
		return false, errors.Wrap(err, "could not get attested header beacon")
	}

	// Compare presence of relevant sync committee
	newHasRelevantSyncCommittee := IsSyncCommitteeUpdate(newUpdate) && (slots.SyncCommitteePeriod(slots.ToEpoch(newUpdateAttestedHeaderBeacon.Slot)) == slots.SyncCommitteePeriod(slots.ToEpoch(newUpdate.SignatureSlot)))
	oldHasRelevantSyncCommittee := IsSyncCommitteeUpdate(oldUpdate) && (slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdateAttestedHeaderBeacon.Slot)) == slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdate.SignatureSlot)))

	if newHasRelevantSyncCommittee != oldHasRelevantSyncCommittee {
		return newHasRelevantSyncCommittee, nil
	}

	// Compare indication of any finality
	newHasFinality := IsFinalityUpdate(newUpdate)
	oldHasFinality := IsFinalityUpdate(oldUpdate)
	if newHasFinality != oldHasFinality {
		return newHasFinality, nil
	}

	newUpdateFinalizedHeaderBeacon, err := newUpdate.FinalizedHeader.GetBeacon()
	if err != nil {
		return false, errors.Wrap(err, "could not get finalized header beacon")
	}
	oldUpdateFinalizedHeaderBeacon, err := oldUpdate.FinalizedHeader.GetBeacon()
	if err != nil {
		return false, errors.Wrap(err, "could not get finalized header beacon")
	}

	// Compare sync committee finality
	if newHasFinality {
		newHasSyncCommitteeFinality := slots.SyncCommitteePeriod(slots.ToEpoch(newUpdateFinalizedHeaderBeacon.Slot)) == slots.SyncCommitteePeriod(slots.ToEpoch(newUpdateAttestedHeaderBeacon.Slot))
		oldHasSyncCommitteeFinality := slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdateFinalizedHeaderBeacon.Slot)) == slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdateAttestedHeaderBeacon.Slot))

		if newHasSyncCommitteeFinality != oldHasSyncCommitteeFinality {
			return newHasSyncCommitteeFinality, nil
		}
	}

	// Tiebreaker 1: Sync committee participation beyond supermajority
	if newNumActiveParticipants != oldNumActiveParticipants {
		return newNumActiveParticipants > oldNumActiveParticipants, nil
	}

	// Tiebreaker 2: Prefer older data (fewer changes to best)
	if newUpdateAttestedHeaderBeacon.Slot != oldUpdateAttestedHeaderBeacon.Slot {
		return newUpdateAttestedHeaderBeacon.Slot < oldUpdateAttestedHeaderBeacon.Slot, nil
	}
	return newUpdate.SignatureSlot < oldUpdate.SignatureSlot, nil
}
