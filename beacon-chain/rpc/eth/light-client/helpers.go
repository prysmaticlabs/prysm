package lightclient

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	v1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

const (
	nextSyncCommitteeBranchNumOfLeaves = 5
)

// CreateLightClientBootstrap - implements https://github.com/ethereum/consensus-specs/blob/3d235740e5f1e641d3b160c8688f26e7dc5a1894/specs/altair/light-client/full-node.md#create_light_client_bootstrap
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
func CreateLightClientBootstrap(ctx context.Context, state state.BeaconState) (*LightClientBootstrap, error) {
	// assert compute_epoch_at_slot(state.slot) >= ALTAIR_FORK_EPOCH
	if slots.ToEpoch(state.Slot()) < params.BeaconConfig().AltairForkEpoch {
		return nil, fmt.Errorf("invalid state slot %d", state.Slot())
	}

	// assert state.slot == state.latest_block_header.slot
	if state.Slot() != state.LatestBlockHeader().Slot {
		return nil, fmt.Errorf("invalid state slot %d", state.Slot())
	}

	// Prepare data
	latestBlockHeader := state.LatestBlockHeader()

	currentSyncCommittee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, fmt.Errorf("could not get current sync committee %v", err)
	}

	currentSyncCommitteePubkeys := currentSyncCommittee.GetPubkeys()
	committee := apimiddleware.SyncCommitteeJson{
		Pubkeys:         make([]string, len(currentSyncCommitteePubkeys)),
		AggregatePubkey: hexutil.Encode(currentSyncCommittee.GetAggregatePubkey()),
	}
	for i, pubkey := range currentSyncCommitteePubkeys {
		committee.Pubkeys[i] = hexutil.Encode(pubkey)
	}
	currentSyncCommitteeProof, err := state.CurrentSyncCommitteeProof(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get current sync committee proof %v", err)
	}

	branch := make([]string, len(currentSyncCommitteeProof))
	for i, proof := range currentSyncCommitteeProof {
		branch[i] = hexutil.Encode(proof)
	}

	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get state root %v", err)
	}

	// Return result
	result := &LightClientBootstrap{
		Header: &apimiddleware.BeaconBlockHeaderJson{
			Slot:          strconv.FormatUint(uint64(latestBlockHeader.Slot), 10),
			ProposerIndex: strconv.FormatUint(uint64(latestBlockHeader.ProposerIndex), 10),
			ParentRoot:    hexutil.Encode(latestBlockHeader.ParentRoot),
			StateRoot:     hexutil.Encode(stateRoot[:]),
			BodyRoot:      hexutil.Encode(latestBlockHeader.BodyRoot),
		},
		CurrentSyncCommittee:       &committee,
		CurrentSyncCommitteeBranch: branch,
	}

	return result, nil
}

// CreateLightClientUpdate - implements https://github.
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
func CreateLightClientUpdate(
	ctx context.Context,
	slotsPerPeriod uint64,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock) (*LightClientUpdate, error) {
	result, err := blockchain.NewLightClientFinalityUpdateFromBeaconState(ctx, state, block, attestedState, finalizedBlock)
	if err != nil {
		return nil, err
	}

	// Generate next sync committee and proof
	var nextSyncCommittee *v2.SyncCommittee
	var nextSyncCommitteeBranch [][]byte

	// update_signature_period = compute_sync_committee_period(compute_epoch_at_slot(block.message.slot))
	updateSignaturePeriod := uint64(block.Block().Slot()) / slotsPerPeriod

	// update_attested_period = compute_sync_committee_period(compute_epoch_at_slot(attested_header.slot))
	updateAttestedPeriod := uint64(result.AttestedHeader.Slot) / slotsPerPeriod

	if updateAttestedPeriod == updateSignaturePeriod {
		tempNextSyncCommittee, err := attestedState.NextSyncCommittee()
		if err != nil {
			return nil, fmt.Errorf("could not get next sync committee %v", err)
		}

		nextSyncCommittee = &v2.SyncCommittee{
			Pubkeys:         tempNextSyncCommittee.Pubkeys,
			AggregatePubkey: tempNextSyncCommittee.AggregatePubkey,
		}

		nextSyncCommitteeBranch, err = attestedState.NextSyncCommitteeProof(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not get next sync committee proof %v", err)
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

		nextSyncCommitteeBranch = make([][]byte, nextSyncCommitteeBranchNumOfLeaves)
		for i := 0; i < 5; i++ {
			nextSyncCommitteeBranch[i] = make([]byte, fieldparams.RootLength)
		}
	}

	result.NextSyncCommittee = nextSyncCommittee
	result.NextSyncCommitteeBranch = nextSyncCommitteeBranch
	return NewLightClientUpdateToJSON(result), nil
}

func NewLightClientFinalityUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock) (*LightClientUpdate, error) {
	result, err := blockchain.NewLightClientFinalityUpdateFromBeaconState(ctx, state, block, attestedState, finalizedBlock)
	if err != nil {
		return nil, err
	}

	return NewLightClientUpdateToJSON(result), nil
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

func headerToJSON(input *v1.BeaconBlockHeader) *apimiddleware.BeaconBlockHeaderJson {
	if input == nil {
		return nil
	}
	return &apimiddleware.BeaconBlockHeaderJson{
		Slot:          strconv.FormatUint(uint64(input.Slot), 10),
		ProposerIndex: strconv.FormatUint(uint64(input.ProposerIndex), 10),
		ParentRoot:    hexutil.Encode(input.ParentRoot),
		StateRoot:     hexutil.Encode(input.StateRoot),
		BodyRoot:      hexutil.Encode(input.BodyRoot),
	}
}

func syncCommitteeToJSON(input *v2.SyncCommittee) *apimiddleware.SyncCommitteeJson {
	if input == nil {
		return nil
	}
	syncCommittee := &apimiddleware.SyncCommitteeJson{
		AggregatePubkey: hexutil.Encode(input.AggregatePubkey),
		Pubkeys:         make([]string, len(input.Pubkeys)),
	}
	for i, pubKey := range input.Pubkeys {
		syncCommittee.Pubkeys[i] = hexutil.Encode(pubKey)
	}
	return syncCommittee
}

func syncAggregateToJSON(input *v1.SyncAggregate) *apimiddleware.SyncAggregateJson {
	if input == nil {
		return nil
	}
	return &apimiddleware.SyncAggregateJson{
		SyncCommitteeBits:      hexutil.Encode(input.SyncCommitteeBits),
		SyncCommitteeSignature: hexutil.Encode(input.SyncCommitteeSignature),
	}
}

func NewLightClientUpdateToJSON(input *v2.LightClientUpdate) *LightClientUpdate {
	return &LightClientUpdate{
		AttestedHeader:          headerToJSON(input.AttestedHeader),
		NextSyncCommittee:       syncCommitteeToJSON(input.NextSyncCommittee),
		NextSyncCommitteeBranch: branchToJSON(input.NextSyncCommitteeBranch),
		FinalizedHeader:         headerToJSON(input.FinalizedHeader),
		FinalityBranch:          branchToJSON(input.FinalityBranch),
		SyncAggregate:           syncAggregateToJSON(input.SyncAggregate),
		SignatureSlot:           strconv.FormatUint(uint64(input.SignatureSlot), 10),
	}
}
