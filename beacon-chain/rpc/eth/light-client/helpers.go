package lightclient

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
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
