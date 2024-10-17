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
	light_client "github.com/prysmaticlabs/prysm/v5/consensus-types/light-client"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	v11 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpbv1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	pb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"google.golang.org/protobuf/proto"
)

const (
	FinalityBranchNumOfLeaves = 6
)

func NewLightClientFinalityUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	attestedBlock interfaces.ReadOnlySignedBeaconBlock,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock,
) (interfaces.LightClientFinalityUpdate, error) {
	update, err := NewLightClientUpdateFromBeaconState(ctx, state, block, attestedState, attestedBlock, finalizedBlock)
	if err != nil {
		return nil, err
	}

	return light_client.NewFinalityUpdateFromUpdate(update)
}

func NewLightClientOptimisticUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	attestedBlock interfaces.ReadOnlySignedBeaconBlock,
) (interfaces.LightClientOptimisticUpdate, error) {
	update, err := NewLightClientUpdateFromBeaconState(ctx, state, block, attestedState, attestedBlock, nil)
	if err != nil {
		return nil, err
	}

	return light_client.NewOptimisticUpdateFromUpdate(update)
}

func NewLightClientUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	attestedBlock interfaces.ReadOnlySignedBeaconBlock,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock) (interfaces.LightClientUpdate, error) {
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
		return nil, fmt.Errorf(
			"attested state slot %d not equal to attested latest block header slot %d",
			attestedState.Slot(),
			attestedState.LatestBlockHeader().Slot,
		)
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
		return nil, fmt.Errorf(
			"attested header root %#x not equal to block parent root %#x or attested block root %#x",
			attestedHeaderRoot,
			block.Block().ParentRoot(),
			attestedBlockRoot,
		)
	}

	// update_attested_period = compute_sync_committee_period_at_slot(attested_block.message.slot)
	updateAttestedPeriod := slots.SyncCommitteePeriod(slots.ToEpoch(attestedBlock.Block().Slot()))

	// update = LightClientUpdate()
	result, err := createDefaultLightClientUpdate(attestedBlock.Version())
	if err != nil {
		return nil, errors.Wrap(err, "could not create default light client update")
	}

	// update.attested_header = block_to_light_client_header(attested_block)
	attestedLightClientHeader, err := BlockToLightClientHeader(attestedBlock.Version(), attestedBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attested light client header")
	}
	result.SetAttestedHeader(attestedLightClientHeader)

	// if update_attested_period == update_signature_period
	if updateAttestedPeriod == updateSignaturePeriod {
		// update.next_sync_committee = attested_state.next_sync_committee
		tempNextSyncCommittee, err := attestedState.NextSyncCommittee()
		if err != nil {
			return nil, errors.Wrap(err, "could not get next sync committee")
		}
		nextSyncCommittee := &pb.SyncCommittee{
			Pubkeys:         tempNextSyncCommittee.Pubkeys,
			AggregatePubkey: tempNextSyncCommittee.AggregatePubkey,
		}
		result.SetNextSyncCommittee(nextSyncCommittee)

		// update.next_sync_committee_branch = NextSyncCommitteeBranch(
		//     compute_merkle_proof(attested_state, next_sync_committee_gindex_at_slot(attested_state.slot)))
		nextSyncCommitteeBranch, err := attestedState.NextSyncCommitteeProof(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get next sync committee proof")
		}
		if attestedBlock.Version() >= version.Electra {
			if err = result.SetNextSyncCommitteeBranchElectra(nextSyncCommitteeBranch); err != nil {
				return nil, errors.Wrap(err, "could not set next sync committee branch")
			}
		} else if err = result.SetNextSyncCommitteeBranch(nextSyncCommitteeBranch); err != nil {
			return nil, errors.Wrap(err, "could not set next sync committee branch")
		}
	}

	// if finalized_block is not None
	if finalizedBlock != nil && !finalizedBlock.IsNil() {
		// if finalized_block.message.slot != GENESIS_SLOT
		if finalizedBlock.Block().Slot() != 0 {
			// update.finalized_header = block_to_light_client_header(finalized_block)
			finalizedLightClientHeader, err := BlockToLightClientHeader(attestedBlock.Version(), finalizedBlock)
			if err != nil {
				return nil, errors.Wrap(err, "could not get finalized light client header")
			}
			result.SetFinalizedHeader(finalizedLightClientHeader)
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
		if err = result.SetFinalityBranch(finalityBranch); err != nil {
			return nil, errors.Wrap(err, "could not set finality branch")
		}
	}

	// update.sync_aggregate = block.message.body.sync_aggregate
	result.SetSyncAggregate(&pb.SyncAggregate{
		SyncCommitteeBits:      syncAggregate.SyncCommitteeBits,
		SyncCommitteeSignature: syncAggregate.SyncCommitteeSignature,
	})

	// update.signature_slot = block.message.slot
	result.SetSignatureSlot(block.Block().Slot())

	return result, nil
}

func createDefaultLightClientUpdate(ver int) (interfaces.LightClientUpdate, error) {
	syncCommitteeSize := params.BeaconConfig().SyncCommitteeSize
	pubKeys := make([][]byte, syncCommitteeSize)
	for i := uint64(0); i < syncCommitteeSize; i++ {
		pubKeys[i] = make([]byte, fieldparams.BLSPubkeyLength)
	}
	nextSyncCommittee := &pb.SyncCommittee{
		Pubkeys:         pubKeys,
		AggregatePubkey: make([]byte, fieldparams.BLSPubkeyLength),
	}

	var scDepth int
	if ver >= version.Electra {
		scDepth = fieldparams.SyncCommitteeBranchDepthElectra
	} else {
		scDepth = fieldparams.SyncCommitteeBranchDepth
	}
	nextSyncCommitteeBranch := make([][]byte, scDepth)
	for i := 0; i < scDepth; i++ {
		nextSyncCommitteeBranch[i] = make([]byte, fieldparams.RootLength)
	}

	executionBranch := make([][]byte, fieldparams.ExecutionBranchDepth)
	for i := 0; i < fieldparams.ExecutionBranchDepth; i++ {
		executionBranch[i] = make([]byte, 32)
	}
	finalityBranch := make([][]byte, fieldparams.FinalityBranchDepth)
	for i := 0; i < fieldparams.FinalityBranchDepth; i++ {
		finalityBranch[i] = make([]byte, 32)
	}

	var update proto.Message
	switch ver {
	case version.Altair, version.Bellatrix:
		update = &pb.LightClientUpdateAltair{
			NextSyncCommittee:       nextSyncCommittee,
			NextSyncCommitteeBranch: nextSyncCommitteeBranch,
			FinalityBranch:          finalityBranch,
		}
	case version.Capella:
		update = &pb.LightClientUpdateCapella{
			NextSyncCommittee:       nextSyncCommittee,
			NextSyncCommitteeBranch: nextSyncCommitteeBranch,
			FinalityBranch:          finalityBranch,
		}
	case version.Deneb:
		update = &pb.LightClientUpdateDeneb{
			NextSyncCommittee:       nextSyncCommittee,
			NextSyncCommitteeBranch: nextSyncCommitteeBranch,
			FinalityBranch:          finalityBranch,
		}
	case version.Electra:
		update = &pb.LightClientUpdateElectra{
			NextSyncCommittee:       nextSyncCommittee,
			NextSyncCommitteeBranch: nextSyncCommitteeBranch,
			FinalityBranch:          finalityBranch,
		}
	default:
		return nil, fmt.Errorf("unsupported version %s", version.String(ver))
	}

	return light_client.NewWrappedUpdate(update)
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

func BlockToLightClientHeader(ver int, block interfaces.ReadOnlySignedBeaconBlock) (interfaces.LightClientHeader, error) {
	var header proto.Message
	var err error

	switch ver {
	case version.Altair, version.Bellatrix:
		header, err = blockToLightClientHeaderAltair(block)
		if err != nil {
			return nil, errors.Wrap(err, "could not get header")
		}
	case version.Capella:
		header, err = blockToLightClientHeaderCapella(context.Background(), block)
		if err != nil {
			return nil, errors.Wrap(err, "could not get header")
		}
	case version.Deneb, version.Electra:
		header, err = blockToLightClientHeaderDeneb(context.Background(), block)
		if err != nil {
			return nil, errors.Wrap(err, "could not get header")
		}
	default:
		return nil, fmt.Errorf("unsupported block version %s", version.String(ver))
	}

	return light_client.NewWrappedHeader(header)
}

func blockToLightClientHeaderAltair(block interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.LightClientHeader, error) {
	if block.Version() < version.Altair {
		return nil, fmt.Errorf("block version %s is before Altair", version.String(block.Version()))
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
		return nil, fmt.Errorf("block version %s is before Capella", version.String(block.Version()))
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
		return nil, fmt.Errorf("block version %s is before Deneb", version.String(block.Version()))
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
