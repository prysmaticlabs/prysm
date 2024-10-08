package light_client

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	v11 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpbv1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

const (
	FinalityBranchNumOfLeaves  = 6
	executionBranchNumOfLeaves = 4
)

func createLightClientFinalityUpdate(update *ethpbv2.LightClientUpdate) *ethpbv2.LightClientFinalityUpdate {
	finalityUpdate := &ethpbv2.LightClientFinalityUpdate{
		AttestedHeader:  update.AttestedHeader,
		FinalizedHeader: update.FinalizedHeader,
		FinalityBranch:  update.FinalityBranch,
		SyncAggregate:   update.SyncAggregate,
		SignatureSlot:   update.SignatureSlot,
	}

	return finalityUpdate
}

func createLightClientOptimisticUpdate(update *ethpbv2.LightClientUpdate) *ethpbv2.LightClientOptimisticUpdate {
	optimisticUpdate := &ethpbv2.LightClientOptimisticUpdate{
		AttestedHeader: update.AttestedHeader,
		SyncAggregate:  update.SyncAggregate,
		SignatureSlot:  update.SignatureSlot,
	}

	return optimisticUpdate
}

func NewLightClientFinalityUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	attestedBlock interfaces.ReadOnlySignedBeaconBlock,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock,
) (*ethpbv2.LightClientFinalityUpdate, error) {
	update, err := NewLightClientUpdateFromBeaconState(ctx, state, block, attestedState, attestedBlock, finalizedBlock)
	if err != nil {
		return nil, err
	}

	return createLightClientFinalityUpdate(update), nil
}

func NewLightClientOptimisticUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	attestedBlock interfaces.ReadOnlySignedBeaconBlock,
) (*ethpbv2.LightClientOptimisticUpdate, error) {
	update, err := NewLightClientUpdateFromBeaconState(ctx, state, block, attestedState, attestedBlock, nil)
	if err != nil {
		return nil, err
	}

	return createLightClientOptimisticUpdate(update), nil
}

func NewLightClientUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	attestedBlock interfaces.ReadOnlySignedBeaconBlock,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.LightClientUpdate, error) {
	// assert compute_epoch_at_slot(attested_state.slot) >= ALTAIR_FORK_EPOCH
	attestedEpoch := slots.ToEpoch(attestedState.Slot())
	if attestedEpoch < params.BeaconConfig().AltairForkEpoch {
		return nil, fmt.Errorf("invalid attested epoch %d", attestedEpoch)
	}

	// assert sum(block.message.body.sync_aggregate.sync_committee_bits) >= MIN_SYNC_COMMITTEE_PARTICIPANTS
	syncAggregate, err := block.Block().Body().SyncAggregate()
	if err != nil {
		return nil, errors.Wrap(err, "could not get sync aggregate")
	}
	if syncAggregate.SyncCommitteeBits.Count() < params.BeaconConfig().MinSyncCommitteeParticipants {
		return nil, fmt.Errorf("invalid sync committee bits count %d", syncAggregate.SyncCommitteeBits.Count())
	}

	// assert state.slot == state.latest_block_header.slot
	if state.Slot() != state.LatestBlockHeader().Slot {
		return nil, fmt.Errorf("state slot %d not equal to latest block header slot %d", state.Slot(), state.LatestBlockHeader().Slot)
	}

	// assert hash_tree_root(header) == hash_tree_root(block.message)
	header := state.LatestBlockHeader()
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state root")
	}
	header.StateRoot = stateRoot[:]
	headerRoot, err := header.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get header root")
	}
	blockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get block root")
	}
	if headerRoot != blockRoot {
		return nil, fmt.Errorf("header root %#x not equal to block root %#x", headerRoot, blockRoot)
	}

	// update_signature_period = compute_sync_committee_period(compute_epoch_at_slot(block.message.slot))
	updateSignaturePeriod := slots.SyncCommitteePeriod(slots.ToEpoch(block.Block().Slot()))

	// assert attested_state.slot == attested_state.latest_block_header.slot
	if attestedState.Slot() != attestedState.LatestBlockHeader().Slot {
		return nil, fmt.Errorf("attested state slot %d not equal to attested latest block header slot %d", attestedState.Slot(), attestedState.LatestBlockHeader().Slot)
	}

	// attested_header = attested_state.latest_block_header.copy()
	attestedHeader := attestedState.LatestBlockHeader()

	// attested_header.state_root = hash_tree_root(attested_state)
	attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attested state root")
	}
	attestedHeader.StateRoot = attestedStateRoot[:]

	// assert hash_tree_root(attested_header) == block.message.parent_root
	attestedHeaderRoot, err := attestedHeader.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get attested header root")
	}
	attestedBlockRoot, err := attestedBlock.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get attested block root")
	}
	// assert hash_tree_root(attested_header) == hash_tree_root(attested_block.message) == block.message.parent_root
	if attestedHeaderRoot != block.Block().ParentRoot() || attestedHeaderRoot != attestedBlockRoot {
		return nil, fmt.Errorf("attested header root %#x not equal to block parent root %#x or attested block root %#x", attestedHeaderRoot, block.Block().ParentRoot(), attestedBlockRoot)
	}

	// update_attested_period = compute_sync_committee_period_at_slot(attested_block.message.slot)
	updateAttestedPeriod := slots.SyncCommitteePeriod(slots.ToEpoch(attestedBlock.Block().Slot()))

	// update = LightClientUpdate()
	result, err := createDefaultLightClientUpdate()
	if err != nil {
		return nil, errors.Wrap(err, "could not create default light client update")
	}

	// update.attested_header = block_to_light_client_header(attested_block)
	attestedLightClientHeader, err := BlockToLightClientHeader(attestedBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attested light client header")
	}
	result.AttestedHeader = attestedLightClientHeader

	// if update_attested_period == update_signature_period
	if updateAttestedPeriod == updateSignaturePeriod {
		tempNextSyncCommittee, err := attestedState.NextSyncCommittee()
		if err != nil {
			return nil, errors.Wrap(err, "could not get next sync committee")
		}
		nextSyncCommittee := &ethpbv2.SyncCommittee{
			Pubkeys:         tempNextSyncCommittee.Pubkeys,
			AggregatePubkey: tempNextSyncCommittee.AggregatePubkey,
		}
		nextSyncCommitteeBranch, err := attestedState.NextSyncCommitteeProof(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get next sync committee proof")
		}

		// update.next_sync_committee = attested_state.next_sync_committee
		result.NextSyncCommittee = nextSyncCommittee

		// update.next_sync_committee_branch = NextSyncCommitteeBranch(
		//     compute_merkle_proof(attested_state, next_sync_committee_gindex_at_slot(attested_state.slot)))
		result.NextSyncCommitteeBranch = nextSyncCommitteeBranch
	}

	// if finalized_block is not None
	if finalizedBlock != nil && !finalizedBlock.IsNil() {
		// if finalized_block.message.slot != GENESIS_SLOT
		if finalizedBlock.Block().Slot() != 0 {
			// update.finalized_header = block_to_light_client_header(finalized_block)
			finalizedLightClientHeader, err := BlockToLightClientHeader(finalizedBlock)
			if err != nil {
				return nil, errors.Wrap(err, "could not get finalized light client header")
			}
			result.FinalizedHeader = finalizedLightClientHeader
		} else {
			// assert attested_state.finalized_checkpoint.root == Bytes32()
			if !bytes.Equal(attestedState.FinalizedCheckpoint().Root, make([]byte, 32)) {
				return nil, fmt.Errorf("invalid finalized header root %v", attestedState.FinalizedCheckpoint().Root)
			}
		}

		// update.finality_branch = FinalityBranch(
		//     compute_merkle_proof(attested_state, finalized_root_gindex_at_slot(attested_state.slot)))
		finalityBranch, err := attestedState.FinalizedRootProof(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get finalized root proof")
		}
		result.FinalityBranch = finalityBranch
	}

	// update.sync_aggregate = block.message.body.sync_aggregate
	result.SyncAggregate = &ethpbv1.SyncAggregate{
		SyncCommitteeBits:      syncAggregate.SyncCommitteeBits,
		SyncCommitteeSignature: syncAggregate.SyncCommitteeSignature,
	}

	// update.signature_slot = block.message.slot
	result.SignatureSlot = block.Block().Slot()

	return result, nil
}

func createDefaultLightClientUpdate() (*ethpbv2.LightClientUpdate, error) {
	syncCommitteeSize := params.BeaconConfig().SyncCommitteeSize
	pubKeys := make([][]byte, syncCommitteeSize)
	for i := uint64(0); i < syncCommitteeSize; i++ {
		pubKeys[i] = make([]byte, fieldparams.BLSPubkeyLength)
	}
	nextSyncCommittee := &ethpbv2.SyncCommittee{
		Pubkeys:         pubKeys,
		AggregatePubkey: make([]byte, fieldparams.BLSPubkeyLength),
	}
	nextSyncCommitteeBranch := make([][]byte, fieldparams.SyncCommitteeBranchDepth)
	for i := 0; i < fieldparams.SyncCommitteeBranchDepth; i++ {
		nextSyncCommitteeBranch[i] = make([]byte, fieldparams.RootLength)
	}
	executionBranch := make([][]byte, executionBranchNumOfLeaves)
	for i := 0; i < executionBranchNumOfLeaves; i++ {
		executionBranch[i] = make([]byte, 32)
	}
	finalityBranch := make([][]byte, FinalityBranchNumOfLeaves)
	for i := 0; i < FinalityBranchNumOfLeaves; i++ {
		finalityBranch[i] = make([]byte, 32)
	}

	return &ethpbv2.LightClientUpdate{
		NextSyncCommittee:       nextSyncCommittee,
		NextSyncCommitteeBranch: nextSyncCommitteeBranch,
		FinalityBranch:          finalityBranch,
	}, nil
}

func ComputeTransactionsRoot(payload interfaces.ExecutionData) ([]byte, error) {
	transactionsRoot, err := payload.TransactionsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		transactions, err := payload.Transactions()
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
	return transactionsRoot, nil
}

func ComputeWithdrawalsRoot(payload interfaces.ExecutionData) ([]byte, error) {
	withdrawalsRoot, err := payload.WithdrawalsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		withdrawals, err := payload.Withdrawals()
		if err != nil {
			return nil, errors.Wrap(err, "could not get withdrawals")
		}
		withdrawalsRootArray, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		if err != nil {
			return nil, errors.Wrap(err, "could not get withdrawals root")
		}
		withdrawalsRoot = withdrawalsRootArray[:]
	} else if err != nil {
		return nil, errors.Wrap(err, "could not get withdrawals root")
	}
	return withdrawalsRoot, nil
}

func BlockToLightClientHeader(block interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.LightClientHeaderContainer, error) {
	switch block.Version() {
	case version.Altair, version.Bellatrix:
		altairHeader, err := blockToLightClientHeaderAltair(block)
		if err != nil {
			return nil, errors.Wrap(err, "could not get header")
		}
		return &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
				HeaderAltair: altairHeader,
			},
		}, nil
	case version.Capella:
		capellaHeader, err := blockToLightClientHeaderCapella(context.Background(), block)
		if err != nil {
			return nil, errors.Wrap(err, "could not get capella header")
		}
		return &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
				HeaderCapella: capellaHeader,
			},
		}, nil
	case version.Deneb, version.Electra:
		denebHeader, err := blockToLightClientHeaderDeneb(context.Background(), block)
		if err != nil {
			return nil, errors.Wrap(err, "could not get header")
		}
		return &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderDeneb{
				HeaderDeneb: denebHeader,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported block version %s", version.String(block.Version()))
	}
}

func blockToLightClientHeaderAltair(block interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.LightClientHeader, error) {
	if block.Version() < version.Altair {
		return nil, fmt.Errorf("block version is %s instead of Altair", version.String(block.Version()))
	}

	parentRoot := block.Block().ParentRoot()
	stateRoot := block.Block().StateRoot()
	bodyRoot, err := block.Block().Body().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get body root")
	}

	return &ethpbv2.LightClientHeader{
		Beacon: &ethpbv1.BeaconBlockHeader{
			Slot:          block.Block().Slot(),
			ProposerIndex: block.Block().ProposerIndex(),
			ParentRoot:    parentRoot[:],
			StateRoot:     stateRoot[:],
			BodyRoot:      bodyRoot[:],
		},
	}, nil
}

func blockToLightClientHeaderCapella(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.LightClientHeaderCapella, error) {
	if block.Version() < version.Capella {
		return nil, fmt.Errorf("block version is %s instead of Capella", version.String(block.Version()))
	}

	payload, err := block.Block().Body().Execution()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload")
	}

	transactionsRoot, err := ComputeTransactionsRoot(payload)
	if err != nil {
		return nil, err
	}
	withdrawalsRoot, err := ComputeWithdrawalsRoot(payload)
	if err != nil {
		return nil, err
	}

	executionHeader := &v11.ExecutionPayloadHeaderCapella{
		ParentHash:       payload.ParentHash(),
		FeeRecipient:     payload.FeeRecipient(),
		StateRoot:        payload.StateRoot(),
		ReceiptsRoot:     payload.ReceiptsRoot(),
		LogsBloom:        payload.LogsBloom(),
		PrevRandao:       payload.PrevRandao(),
		BlockNumber:      payload.BlockNumber(),
		GasLimit:         payload.GasLimit(),
		GasUsed:          payload.GasUsed(),
		Timestamp:        payload.Timestamp(),
		ExtraData:        payload.ExtraData(),
		BaseFeePerGas:    payload.BaseFeePerGas(),
		BlockHash:        payload.BlockHash(),
		TransactionsRoot: transactionsRoot,
		WithdrawalsRoot:  withdrawalsRoot,
	}

	executionPayloadProof, err := blocks.PayloadProof(ctx, block.Block())
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload proof")
	}

	parentRoot := block.Block().ParentRoot()
	stateRoot := block.Block().StateRoot()
	bodyRoot, err := block.Block().Body().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get body root")
	}

	return &ethpbv2.LightClientHeaderCapella{
		Beacon: &ethpbv1.BeaconBlockHeader{
			Slot:          block.Block().Slot(),
			ProposerIndex: block.Block().ProposerIndex(),
			ParentRoot:    parentRoot[:],
			StateRoot:     stateRoot[:],
			BodyRoot:      bodyRoot[:],
		},
		Execution:       executionHeader,
		ExecutionBranch: executionPayloadProof,
	}, nil
}

func blockToLightClientHeaderDeneb(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.LightClientHeaderDeneb, error) {
	if block.Version() < version.Deneb {
		return nil, fmt.Errorf("block version is %s instead of Deneb/Electra", version.String(block.Version()))
	}

	payload, err := block.Block().Body().Execution()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload")
	}

	transactionsRoot, err := ComputeTransactionsRoot(payload)
	if err != nil {
		return nil, err
	}
	withdrawalsRoot, err := ComputeWithdrawalsRoot(payload)
	if err != nil {
		return nil, err
	}
	blobGasUsed, err := payload.BlobGasUsed()
	if err != nil {
		return nil, errors.Wrap(err, "could not get blob gas used")
	}
	excessBlobGas, err := payload.ExcessBlobGas()
	if err != nil {
		return nil, errors.Wrap(err, "could not get excess blob gas")
	}

	executionHeader := &v11.ExecutionPayloadHeaderDeneb{
		ParentHash:       payload.ParentHash(),
		FeeRecipient:     payload.FeeRecipient(),
		StateRoot:        payload.StateRoot(),
		ReceiptsRoot:     payload.ReceiptsRoot(),
		LogsBloom:        payload.LogsBloom(),
		PrevRandao:       payload.PrevRandao(),
		BlockNumber:      payload.BlockNumber(),
		GasLimit:         payload.GasLimit(),
		GasUsed:          payload.GasUsed(),
		Timestamp:        payload.Timestamp(),
		ExtraData:        payload.ExtraData(),
		BaseFeePerGas:    payload.BaseFeePerGas(),
		BlockHash:        payload.BlockHash(),
		TransactionsRoot: transactionsRoot,
		WithdrawalsRoot:  withdrawalsRoot,
		BlobGasUsed:      blobGasUsed,
		ExcessBlobGas:    excessBlobGas,
	}

	executionPayloadProof, err := blocks.PayloadProof(ctx, block.Block())
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload proof")
	}

	parentRoot := block.Block().ParentRoot()
	stateRoot := block.Block().StateRoot()
	bodyRoot, err := block.Block().Body().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get body root")
	}

	return &ethpbv2.LightClientHeaderDeneb{
		Beacon: &ethpbv1.BeaconBlockHeader{
			Slot:          block.Block().Slot(),
			ProposerIndex: block.Block().ProposerIndex(),
			ParentRoot:    parentRoot[:],
			StateRoot:     stateRoot[:],
			BodyRoot:      bodyRoot[:],
		},
		Execution:       executionHeader,
		ExecutionBranch: executionPayloadProof,
	}, nil
}
