package lightclient

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	lightclient "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/light-client"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/proto/migration"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

// createLightClientBootstrap - implements https://github.com/ethereum/consensus-specs/blob/3d235740e5f1e641d3b160c8688f26e7dc5a1894/specs/altair/light-client/full-node.md#create_light_client_bootstrap
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
func createLightClientBootstrap(ctx context.Context, state state.BeaconState) (*structs.LightClientBootstrap, error) {
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
		return nil, fmt.Errorf("could not get current sync committee: %s", err.Error())
	}

	committee := structs.SyncCommitteeFromConsensus(currentSyncCommittee)

	currentSyncCommitteeProof, err := state.CurrentSyncCommitteeProof(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get current sync committee proof: %s", err.Error())
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
		return nil, fmt.Errorf("could not get state root: %s", err.Error())
	}
	header.Beacon.StateRoot = hexutil.Encode(stateRoot[:])

	// Return result
	result := &structs.LightClientBootstrap{
		Header:                     header,
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
	updateAttestedPeriod := slots.SyncCommitteePeriod(slots.ToEpoch(result.AttestedHeader.Beacon.Slot))

	if updateAttestedPeriod == updateSignaturePeriod {
		tempNextSyncCommittee, err := attestedState.NextSyncCommittee()
		if err != nil {
			return nil, fmt.Errorf("could not get next sync committee: %s", err.Error())
		}

		nextSyncCommittee = &v2.SyncCommittee{
			Pubkeys:         tempNextSyncCommittee.Pubkeys,
			AggregatePubkey: tempNextSyncCommittee.AggregatePubkey,
		}

		nextSyncCommitteeBranch, err = attestedState.NextSyncCommitteeProof(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not get next sync committee proof: %s", err.Error())
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
	return newLightClientUpdateToJSON(result), nil
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

	return newLightClientUpdateToJSON(result), nil
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

	return newLightClientUpdateToJSON(result), nil
}

func NewLightClientBootstrapFromJSON(bootstrapJSON *structs.LightClientBootstrap) (*v2.LightClientBootstrap, error) {
	bootstrap := &v2.LightClientBootstrap{}

	var err error

	v1Alpha1Header, err := bootstrapJSON.Header.Beacon.ToConsensus()
	if err != nil {
		return nil, err
	}
	bootstrap.Header = &v2.LightClientHeader{Beacon: migration.V1Alpha1HeaderToV1(v1Alpha1Header)}

	currentSyncCommittee, err := bootstrapJSON.CurrentSyncCommittee.ToConsensus()
	if err != nil {
		return nil, err
	}
	bootstrap.CurrentSyncCommittee = migration.V1Alpha1SyncCommitteeToV2(currentSyncCommittee)

	if bootstrap.CurrentSyncCommitteeBranch, err = branchFromJSON(bootstrapJSON.CurrentSyncCommitteeBranch); err != nil {
		return nil, err
	}
	return bootstrap, nil
}

func branchFromJSON(branch []string) ([][]byte, error) {
	var branchBytes [][]byte
	for _, root := range branch {
		branch, err := hexutil.Decode(root)
		if err != nil {
			return nil, err
		}
		branchBytes = append(branchBytes, branch)
	}
	return branchBytes, nil
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

func newLightClientUpdateToJSON(input *v2.LightClientUpdate) *structs.LightClientUpdate {
	if input == nil {
		return nil
	}

	var nextSyncCommittee *structs.SyncCommittee
	if input.NextSyncCommittee != nil {
		nextSyncCommittee = structs.SyncCommitteeFromConsensus(migration.V2SyncCommitteeToV1Alpha1(input.NextSyncCommittee))
	}

	var finalizedHeader *structs.BeaconBlockHeader
	if input.FinalizedHeader != nil {
		finalizedHeader = structs.BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(input.FinalizedHeader.Beacon))
	}

	return &structs.LightClientUpdate{
		AttestedHeader:          &structs.LightClientHeader{Beacon: structs.BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(input.AttestedHeader.Beacon))},
		NextSyncCommittee:       nextSyncCommittee,
		NextSyncCommitteeBranch: branchToJSON(input.NextSyncCommitteeBranch),
		FinalizedHeader:         &structs.LightClientHeader{Beacon: finalizedHeader},
		FinalityBranch:          branchToJSON(input.FinalityBranch),
		SyncAggregate:           syncAggregateToJSON(input.SyncAggregate),
		SignatureSlot:           strconv.FormatUint(uint64(input.SignatureSlot), 10),
	}
}

func IsSyncCommitteeUpdate(update *v2.LightClientUpdate) bool {
	nextSyncCommitteeBranch := make([][]byte, fieldparams.NextSyncCommitteeBranchDepth)
	return !reflect.DeepEqual(update.NextSyncCommitteeBranch, nextSyncCommitteeBranch)
}

func IsFinalityUpdate(update *v2.LightClientUpdate) bool {
	finalityBranch := make([][]byte, lightclient.FinalityBranchNumOfLeaves)
	return !reflect.DeepEqual(update.FinalityBranch, finalityBranch)
}

func IsBetterUpdate(newUpdate, oldUpdate *v2.LightClientUpdate) bool {
	newNumActiveParticipants := newUpdate.SyncAggregate.SyncCommitteeBits.Count()
	oldNumActiveParticipants := oldUpdate.SyncAggregate.SyncCommitteeBits.Count()
	newHasSupermajority := newUpdate.HasSupermajority()
	oldHasSupermajority := oldUpdate.HasSupermajority()

	if newHasSupermajority != oldHasSupermajority {
		return newHasSupermajority
	}
	if !newHasSupermajority && newNumActiveParticipants != oldNumActiveParticipants {
		return newNumActiveParticipants > oldNumActiveParticipants
	}

	// Compare presence of relevant sync committee
	newHasRelevantSyncCommittee := IsSyncCommitteeUpdate(newUpdate) && (slots.SyncCommitteePeriod(slots.ToEpoch(newUpdate.AttestedHeader.Beacon.Slot)) == slots.SyncCommitteePeriod(slots.ToEpoch(newUpdate.SignatureSlot)))
	oldHasRelevantSyncCommittee := IsSyncCommitteeUpdate(oldUpdate) && (slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdate.AttestedHeader.Beacon.Slot)) == slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdate.SignatureSlot)))

	if newHasRelevantSyncCommittee != oldHasRelevantSyncCommittee {
		return newHasRelevantSyncCommittee
	}

	// Compare indication of any finality
	newHasFinality := IsFinalityUpdate(newUpdate)
	oldHasFinality := IsFinalityUpdate(oldUpdate)
	if newHasFinality != oldHasFinality {
		return newHasFinality
	}

	// Compare sync committee finality
	if newHasFinality {
		newHasSyncCommitteeFinality := slots.SyncCommitteePeriod(slots.ToEpoch(newUpdate.FinalizedHeader.Beacon.Slot)) == slots.SyncCommitteePeriod(slots.ToEpoch(newUpdate.AttestedHeader.Beacon.Slot))
		oldHasSyncCommitteeFinality := slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdate.FinalizedHeader.Beacon.Slot)) == slots.SyncCommitteePeriod(slots.ToEpoch(oldUpdate.AttestedHeader.Beacon.Slot))

		if newHasSyncCommitteeFinality != oldHasSyncCommitteeFinality {
			return newHasSyncCommitteeFinality
		}
	}

	// Tiebreaker 1: Sync committee participation beyond supermajority
	if newNumActiveParticipants != oldNumActiveParticipants {
		return newNumActiveParticipants > oldNumActiveParticipants
	}

	// Tiebreaker 2: Prefer older data (fewer changes to best)
	if newUpdate.AttestedHeader.Beacon.Slot != oldUpdate.AttestedHeader.Beacon.Slot {
		return newUpdate.AttestedHeader.Beacon.Slot < oldUpdate.AttestedHeader.Beacon.Slot
	}
	return newUpdate.SignatureSlot < oldUpdate.SignatureSlot
}
