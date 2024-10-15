package lightclient

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/proto/migration"

	lightclient "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/light-client"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	v2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func createLightClientBootstrap(ctx context.Context, state state.BeaconState, blk interfaces.ReadOnlySignedBeaconBlock) (*structs.LightClientBootstrap, error) {
	switch blk.Version() {
	case version.Phase0:
		return nil, fmt.Errorf("light client bootstrap is not supported for phase0")
	case version.Altair, version.Bellatrix:
		return createLightClientBootstrapAltair(ctx, state, blk)
	case version.Capella:
		return createLightClientBootstrapCapella(ctx, state, blk)
	case version.Deneb, version.Electra:
		return createLightClientBootstrapDeneb(ctx, state, blk)
	}
	return nil, fmt.Errorf("unsupported block version %s", version.String(blk.Version()))
}

func createLightClientBootstrapAltair(ctx context.Context, state state.BeaconState, block interfaces.ReadOnlySignedBeaconBlock) (*structs.LightClientBootstrap, error) {
	// assert compute_epoch_at_slot(state.slot) >= ALTAIR_FORK_EPOCH
	if slots.ToEpoch(state.Slot()) < params.BeaconConfig().AltairForkEpoch {
		return nil, fmt.Errorf("light client bootstrap is not supported before Altair, invalid slot %d", state.Slot())
	}

	// assert state.slot == state.latest_block_header.slot
	latestBlockHeader := state.LatestBlockHeader()
	if state.Slot() != latestBlockHeader.Slot {
		return nil, fmt.Errorf("state slot %d not equal to latest block header slot %d", state.Slot(), latestBlockHeader.Slot)
	}

	// header.state_root = hash_tree_root(state)
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state root")
	}
	latestBlockHeader.StateRoot = stateRoot[:]

	// assert hash_tree_root(header) == hash_tree_root(block.message)
	latestBlockHeaderRoot, err := latestBlockHeader.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get latest block header root")
	}
	beaconBlockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get block root")
	}
	if latestBlockHeaderRoot != beaconBlockRoot {
		return nil, fmt.Errorf("latest block header root %#x not equal to block root %#x", latestBlockHeaderRoot, beaconBlockRoot)
	}

	lightClientHeaderContainer, err := lightclient.BlockToLightClientHeader(block)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert block to light client header")
	}
	lightClientHeader := lightClientHeaderContainer.GetHeaderAltair()

	apiLightClientHeader := &structs.LightClientHeader{
		Beacon: structs.BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(lightClientHeader.Beacon)),
	}

	headerJSON, err := json.Marshal(apiLightClientHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert header to raw message")
	}
	currentSyncCommittee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}
	currentSyncCommitteeProof, err := state.CurrentSyncCommitteeProof(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee proof")
	}

	branch := make([]string, fieldparams.SyncCommitteeBranchDepth)
	for i, proof := range currentSyncCommitteeProof {
		branch[i] = hexutil.Encode(proof)
	}
	result := &structs.LightClientBootstrap{
		Header:                     headerJSON,
		CurrentSyncCommittee:       structs.SyncCommitteeFromConsensus(currentSyncCommittee),
		CurrentSyncCommitteeBranch: branch,
	}

	return result, nil
}

func createLightClientBootstrapCapella(ctx context.Context, state state.BeaconState, block interfaces.ReadOnlySignedBeaconBlock) (*structs.LightClientBootstrap, error) {
	// assert compute_epoch_at_slot(state.slot) >= CAPELLA_FORK_EPOCH
	if slots.ToEpoch(state.Slot()) < params.BeaconConfig().CapellaForkEpoch {
		return nil, fmt.Errorf("creating Capella light client bootstrap is not supported before Capella, invalid slot %d", state.Slot())
	}

	// assert state.slot == state.latest_block_header.slot
	latestBlockHeader := state.LatestBlockHeader()
	if state.Slot() != latestBlockHeader.Slot {
		return nil, fmt.Errorf("state slot %d not equal to latest block header slot %d", state.Slot(), latestBlockHeader.Slot)
	}

	// header.state_root = hash_tree_root(state)
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state root")
	}
	latestBlockHeader.StateRoot = stateRoot[:]

	// assert hash_tree_root(header) == hash_tree_root(block.message)
	latestBlockHeaderRoot, err := latestBlockHeader.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get latest block header root")
	}
	beaconBlockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get block root")
	}
	if latestBlockHeaderRoot != beaconBlockRoot {
		return nil, fmt.Errorf("latest block header root %#x not equal to block root %#x", latestBlockHeaderRoot, beaconBlockRoot)
	}

	lightClientHeaderContainer, err := lightclient.BlockToLightClientHeader(block)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert block to light client header")
	}
	lightClientHeader := lightClientHeaderContainer.GetHeaderCapella()

	apiLightClientHeader := &structs.LightClientHeader{
		Beacon: structs.BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(lightClientHeader.Beacon)),
	}

	headerJSON, err := json.Marshal(apiLightClientHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert header to raw message")
	}
	currentSyncCommittee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}
	currentSyncCommitteeProof, err := state.CurrentSyncCommitteeProof(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee proof")
	}

	branch := make([]string, fieldparams.SyncCommitteeBranchDepth)
	for i, proof := range currentSyncCommitteeProof {
		branch[i] = hexutil.Encode(proof)
	}
	result := &structs.LightClientBootstrap{
		Header:                     headerJSON,
		CurrentSyncCommittee:       structs.SyncCommitteeFromConsensus(currentSyncCommittee),
		CurrentSyncCommitteeBranch: branch,
	}

	return result, nil
}

func createLightClientBootstrapDeneb(ctx context.Context, state state.BeaconState, block interfaces.ReadOnlySignedBeaconBlock) (*structs.LightClientBootstrap, error) {
	// assert compute_epoch_at_slot(state.slot) >= DENEB_FORK_EPOCH
	if slots.ToEpoch(state.Slot()) < params.BeaconConfig().DenebForkEpoch {
		return nil, fmt.Errorf("creating Deneb light client bootstrap is not supported before Deneb, invalid slot %d", state.Slot())
	}

	// assert state.slot == state.latest_block_header.slot
	latestBlockHeader := state.LatestBlockHeader()
	if state.Slot() != latestBlockHeader.Slot {
		return nil, fmt.Errorf("state slot %d not equal to latest block header slot %d", state.Slot(), latestBlockHeader.Slot)
	}

	// header.state_root = hash_tree_root(state)
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state root")
	}
	latestBlockHeader.StateRoot = stateRoot[:]

	// assert hash_tree_root(header) == hash_tree_root(block.message)
	latestBlockHeaderRoot, err := latestBlockHeader.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get latest block header root")
	}
	beaconBlockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get block root")
	}
	if latestBlockHeaderRoot != beaconBlockRoot {
		return nil, fmt.Errorf("latest block header root %#x not equal to block root %#x", latestBlockHeaderRoot, beaconBlockRoot)
	}

	lightClientHeaderContainer, err := lightclient.BlockToLightClientHeader(block)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert block to light client header")
	}
	lightClientHeader := lightClientHeaderContainer.GetHeaderDeneb()

	apiLightClientHeader := &structs.LightClientHeader{
		Beacon: structs.BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(lightClientHeader.Beacon)),
	}

	headerJSON, err := json.Marshal(apiLightClientHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert header to raw message")
	}
	currentSyncCommittee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}
	currentSyncCommitteeProof, err := state.CurrentSyncCommitteeProof(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee proof")
	}
	var branch []string
	switch block.Version() {
	case version.Deneb:
		branch = make([]string, fieldparams.SyncCommitteeBranchDepth)
	case version.Electra:
		branch = make([]string, fieldparams.SyncCommitteeBranchDepthElectra)
	}
	for i, proof := range currentSyncCommitteeProof {
		branch[i] = hexutil.Encode(proof)
	}
	result := &structs.LightClientBootstrap{
		Header:                     headerJSON,
		CurrentSyncCommittee:       structs.SyncCommitteeFromConsensus(currentSyncCommittee),
		CurrentSyncCommitteeBranch: branch,
	}

	return result, nil
}

func newLightClientUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	attestedBlock interfaces.ReadOnlySignedBeaconBlock,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock,
) (*structs.LightClientUpdate, error) {
	result, err := lightclient.NewLightClientUpdateFromBeaconState(ctx, state, block, attestedState, attestedBlock, finalizedBlock)
	if err != nil {
		return nil, err
	}

	return structs.LightClientUpdateFromConsensus(result)
}

func newLightClientFinalityUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	attestedBlock interfaces.ReadOnlySignedBeaconBlock,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock,
) (*structs.LightClientFinalityUpdate, error) {
	result, err := lightclient.NewLightClientFinalityUpdateFromBeaconState(ctx, state, block, attestedState, attestedBlock, finalizedBlock)
	if err != nil {
		return nil, err
	}

	return structs.LightClientFinalityUpdateFromConsensus(result)
}

func newLightClientOptimisticUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	attestedBlock interfaces.ReadOnlySignedBeaconBlock,
) (*structs.LightClientOptimisticUpdate, error) {
	result, err := lightclient.NewLightClientOptimisticUpdateFromBeaconState(ctx, state, block, attestedState, attestedBlock)
	if err != nil {
		return nil, err
	}

	return structs.LightClientOptimisticUpdateFromConsensus(result)
}

func IsSyncCommitteeUpdate(update *v2.LightClientUpdate) bool {
	nextSyncCommitteeBranch := make([][]byte, fieldparams.SyncCommitteeBranchDepth)
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
