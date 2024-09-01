package light_client

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpbv1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/proto/migration"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"

	"context"

	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
)

const (
	FinalityBranchNumOfLeaves = 6
)

// CreateLightClientFinalityUpdate - implements https://github.com/ethereum/consensus-specs/blob/3d235740e5f1e641d3b160c8688f26e7dc5a1894/specs/altair/light-client/full-node.md#create_light_client_finality_update
// def create_light_client_finality_update(update: LightClientUpdate) -> LightClientFinalityUpdate:
//
//	return LightClientFinalityUpdate(
//	    attested_header=update.attested_header,
//	    finalized_header=update.finalized_header,
//	    finality_branch=update.finality_branch,
//	    sync_aggregate=update.sync_aggregate,
//	    signature_slot=update.signature_slot,
//	)
func CreateLightClientFinalityUpdate(update *ethpbv2.LightClientUpdate) *ethpbv2.LightClientFinalityUpdate {

	finalityUpdate := &ethpbv2.LightClientFinalityUpdate{
		AttestedHeader:  update.AttestedHeader,
		FinalizedHeader: update.FinalizedHeader,
		FinalityBranch:  update.FinalityBranch,
		SyncAggregate:   update.SyncAggregate,
		SignatureSlot:   update.SignatureSlot,
	}

	return finalityUpdate
}

// CreateLightClientOptimisticUpdate - implements https://github.com/ethereum/consensus-specs/blob/3d235740e5f1e641d3b160c8688f26e7dc5a1894/specs/altair/light-client/full-node.md#create_light_client_optimistic_update
// def create_light_client_optimistic_update(update: LightClientUpdate) -> LightClientOptimisticUpdate:
//
//	return LightClientOptimisticUpdate(
//	    attested_header=update.attested_header,
//	    sync_aggregate=update.sync_aggregate,
//	    signature_slot=update.signature_slot,
//	)
func CreateLightClientOptimisticUpdate(update *ethpbv2.LightClientUpdate) *ethpbv2.LightClientOptimisticUpdate {
	optimisticUpdate := &ethpbv2.LightClientOptimisticUpdate{
		AttestedHeader: update.AttestedHeader,
		SyncAggregate:  update.SyncAggregate,
		SignatureSlot:  update.SignatureSlot,
	}

	return optimisticUpdate
}

func NewLightClientOptimisticUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState) (*ethpbv2.LightClientUpdate, error) {
	// assert compute_epoch_at_slot(attested_state.slot) >= ALTAIR_FORK_EPOCH
	attestedEpoch := slots.ToEpoch(attestedState.Slot())
	if attestedEpoch < params.BeaconConfig().AltairForkEpoch {
		return nil, fmt.Errorf("invalid attested epoch %d", attestedEpoch)
	}

	// assert sum(block.message.body.sync_aggregate.sync_committee_bits) >= MIN_SYNC_COMMITTEE_PARTICIPANTS
	syncAggregate, err := block.Block().Body().SyncAggregate()
	if err != nil {
		return nil, fmt.Errorf("could not get sync aggregate %w", err)
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
		return nil, fmt.Errorf("could not get state root %w", err)
	}
	header.StateRoot = stateRoot[:]

	headerRoot, err := header.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("could not get header root %w", err)
	}

	blockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("could not get block root %w", err)
	}

	if headerRoot != blockRoot {
		return nil, fmt.Errorf("header root %#x not equal to block root %#x", headerRoot, blockRoot)
	}

	// assert attested_state.slot == attested_state.latest_block_header.slot
	if attestedState.Slot() != attestedState.LatestBlockHeader().Slot {
		return nil, fmt.Errorf("attested state slot %d not equal to attested latest block header slot %d", attestedState.Slot(), attestedState.LatestBlockHeader().Slot)
	}

	// attested_header = attested_state.latest_block_header.copy()
	attestedHeader := attestedState.LatestBlockHeader()

	// attested_header.state_root = hash_tree_root(attested_state)
	attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get attested state root %w", err)
	}
	attestedHeader.StateRoot = attestedStateRoot[:]

	// assert hash_tree_root(attested_header) == block.message.parent_root
	attestedHeaderRoot, err := attestedHeader.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("could not get attested header root %w", err)
	}

	if attestedHeaderRoot != block.Block().ParentRoot() {
		return nil, fmt.Errorf("attested header root %#x not equal to block parent root %#x", attestedHeaderRoot, block.Block().ParentRoot())
	}

	syncAggregateResult := &ethpbv1.SyncAggregate{
		SyncCommitteeBits:      syncAggregate.SyncCommitteeBits,
		SyncCommitteeSignature: syncAggregate.SyncCommitteeSignature,
	}

	result := &ethpbv2.LightClientUpdate{
		SyncAggregate: syncAggregateResult,
		SignatureSlot: block.Block().Slot(),
	}

	// Altair block
	if block.Block().Version() == version.Altair || block.Block().Version() == version.Bellatrix {
		result.AttestedHeader = &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
				HeaderAltair: &ethpbv2.LightClientHeader{
					Beacon: &ethpbv1.BeaconBlockHeader{
						Slot:          attestedHeader.Slot,
						ProposerIndex: attestedHeader.ProposerIndex,
						ParentRoot:    attestedHeader.ParentRoot,
						StateRoot:     attestedHeader.StateRoot,
						BodyRoot:      attestedHeader.BodyRoot,
					},
				},
			},
		}
		return result, nil
	}
	// post altair block
	payloadInterface, err := block.Block().Body().Execution()
	if err != nil {
		return nil, fmt.Errorf("could not get execution payload header: %s", err.Error())
	}
	transactionsRoot, err := payloadInterface.TransactionsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		transactions, err := payloadInterface.Transactions()
		if err != nil {
			return nil, fmt.Errorf("could not get transactions: %s", err.Error())
		}
		transactionsRootArray, err := ssz.TransactionsRoot(transactions)
		if err != nil {
			return nil, fmt.Errorf("could not get transactions root: %s", err.Error())
		}
		transactionsRoot = transactionsRootArray[:]
	} else if err != nil {
		return nil, fmt.Errorf("could not get transactions root: %s", err.Error())
	}
	withdrawalsRoot, err := payloadInterface.WithdrawalsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		withdrawals, err := payloadInterface.Withdrawals()
		if err != nil {
			return nil, fmt.Errorf("could not get withdrawals: %s", err.Error())
		}
		withdrawalsRootArray, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		if err != nil {
			return nil, fmt.Errorf("could not get withdrawals root: %s", err.Error())
		}
		withdrawalsRoot = withdrawalsRootArray[:]
	}
	// Capella block
	if block.Block().Version() == version.Capella {
		executionPayloadHeader := &enginev1.ExecutionPayloadHeaderCapella{
			ParentHash:       payloadInterface.ParentHash(),
			FeeRecipient:     payloadInterface.FeeRecipient(),
			StateRoot:        payloadInterface.StateRoot(),
			ReceiptsRoot:     payloadInterface.ReceiptsRoot(),
			LogsBloom:        payloadInterface.LogsBloom(),
			PrevRandao:       payloadInterface.PrevRandao(),
			BlockNumber:      payloadInterface.BlockNumber(),
			GasLimit:         payloadInterface.GasLimit(),
			GasUsed:          payloadInterface.GasUsed(),
			Timestamp:        payloadInterface.Timestamp(),
			ExtraData:        payloadInterface.ExtraData(),
			BaseFeePerGas:    payloadInterface.BaseFeePerGas(),
			BlockHash:        payloadInterface.BlockHash(),
			TransactionsRoot: transactionsRoot,
			WithdrawalsRoot:  withdrawalsRoot,
		}

		executionPayloadProof, err := blocks.PayloadProof(ctx, block.Block())
		if err != nil {
			return nil, fmt.Errorf("could not get execution payload proof: %s", err.Error())
		}

		result.AttestedHeader = &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
				HeaderCapella: &ethpbv2.LightClientHeaderCapella{
					Beacon: &ethpbv1.BeaconBlockHeader{
						Slot:          attestedHeader.Slot,
						ProposerIndex: attestedHeader.ProposerIndex,
						ParentRoot:    attestedHeader.ParentRoot,
						StateRoot:     attestedHeader.StateRoot,
						BodyRoot:      attestedHeader.BodyRoot,
					},
					Execution:       executionPayloadHeader,
					ExecutionBranch: executionPayloadProof,
				},
			},
		}

		return result, nil
	}
	// Deneb block
	if block.Block().Version() == version.Deneb || block.Block().Version() == version.Electra {
		executionPayloadHeader := &enginev1.ExecutionPayloadHeaderDeneb{
			ParentHash:       payloadInterface.ParentHash(),
			FeeRecipient:     payloadInterface.FeeRecipient(),
			StateRoot:        payloadInterface.StateRoot(),
			ReceiptsRoot:     payloadInterface.ReceiptsRoot(),
			LogsBloom:        payloadInterface.LogsBloom(),
			PrevRandao:       payloadInterface.PrevRandao(),
			BlockNumber:      payloadInterface.BlockNumber(),
			GasLimit:         payloadInterface.GasLimit(),
			GasUsed:          payloadInterface.GasUsed(),
			Timestamp:        payloadInterface.Timestamp(),
			ExtraData:        payloadInterface.ExtraData(),
			BaseFeePerGas:    payloadInterface.BaseFeePerGas(),
			BlockHash:        payloadInterface.BlockHash(),
			TransactionsRoot: transactionsRoot,
			WithdrawalsRoot:  withdrawalsRoot,
		}

		executionPayloadProof, err := blocks.PayloadProof(ctx, block.Block())
		if err != nil {
			return nil, fmt.Errorf("could not get execution payload proof: %s", err.Error())
		}

		result.AttestedHeader = &ethpbv2.LightClientHeaderContainer{
			Header: &ethpbv2.LightClientHeaderContainer_HeaderDeneb{
				HeaderDeneb: &ethpbv2.LightClientHeaderDeneb{
					Beacon: &ethpbv1.BeaconBlockHeader{
						Slot:          attestedHeader.Slot,
						ProposerIndex: attestedHeader.ProposerIndex,
						ParentRoot:    attestedHeader.ParentRoot,
						StateRoot:     attestedHeader.StateRoot,
						BodyRoot:      attestedHeader.BodyRoot,
					},
					Execution:       executionPayloadHeader,
					ExecutionBranch: executionPayloadProof,
				},
			},
		}

		return result, nil
	}

	return nil, fmt.Errorf("unsupported block version %d", block.Block().Version())
}

func NewLightClientFinalityUpdateFromBeaconState(
	ctx context.Context,
	state state.BeaconState,
	block interfaces.ReadOnlySignedBeaconBlock,
	attestedState state.BeaconState,
	finalizedBlock interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.LightClientUpdate, error) {
	result, err := NewLightClientOptimisticUpdateFromBeaconState(
		ctx,
		state,
		block,
		attestedState,
	)
	if err != nil {
		return nil, err
	}

	if block.Block().Version() != version.Altair || block.Block().Version() != version.Bellatrix {
		return nil, fmt.Errorf("unsupported block version %d", block.Block().Version())
	}
	// Indicate finality whenever possible
	var finalizedHeader *ethpbv2.LightClientHeader
	var finalityBranch [][]byte

	if finalizedBlock != nil && !finalizedBlock.IsNil() {
		if finalizedBlock.Block().Slot() != 0 {
			tempFinalizedHeader, err := finalizedBlock.Header()
			if err != nil {
				return nil, fmt.Errorf("could not get finalized header %w", err)
			}
			finalizedHeader = &ethpbv2.LightClientHeader{Beacon: migration.V1Alpha1SignedHeaderToV1(tempFinalizedHeader).GetMessage()}

			finalizedHeaderRoot, err := finalizedHeader.Beacon.HashTreeRoot()
			if err != nil {
				return nil, fmt.Errorf("could not get finalized header root %w", err)
			}

			if finalizedHeaderRoot != bytesutil.ToBytes32(attestedState.FinalizedCheckpoint().Root) {
				return nil, fmt.Errorf("finalized header root %#x not equal to attested finalized checkpoint root %#x", finalizedHeaderRoot, bytesutil.ToBytes32(attestedState.FinalizedCheckpoint().Root))
			}
		} else {
			if !bytes.Equal(attestedState.FinalizedCheckpoint().Root, make([]byte, 32)) {
				return nil, fmt.Errorf("invalid finalized header root %v", attestedState.FinalizedCheckpoint().Root)
			}

			finalizedHeader = &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
				Slot:          0,
				ProposerIndex: 0,
				ParentRoot:    make([]byte, 32),
				StateRoot:     make([]byte, 32),
				BodyRoot:      make([]byte, 32),
			}}
		}

		var bErr error
		finalityBranch, bErr = attestedState.FinalizedRootProof(ctx)
		if bErr != nil {
			return nil, fmt.Errorf("could not get finalized root proof %w", bErr)
		}
	} else {
		finalizedHeader = &ethpbv2.LightClientHeader{Beacon: &ethpbv1.BeaconBlockHeader{
			Slot:          0,
			ProposerIndex: 0,
			ParentRoot:    make([]byte, 32),
			StateRoot:     make([]byte, 32),
			BodyRoot:      make([]byte, 32),
		}}

		finalityBranch = make([][]byte, FinalityBranchNumOfLeaves)
		for i := 0; i < FinalityBranchNumOfLeaves; i++ {
			finalityBranch[i] = make([]byte, 32)
		}
	}

	result.FinalizedHeader = &ethpbv2.LightClientHeaderContainer{
		Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
			HeaderAltair: finalizedHeader,
		},
	}
	result.FinalityBranch = finalityBranch
	return result, nil
}

func NewLightClientUpdateFromFinalityUpdate(update *ethpbv2.LightClientFinalityUpdate) *ethpbv2.LightClientUpdate {
	return &ethpbv2.LightClientUpdate{
		AttestedHeader:  update.AttestedHeader,
		FinalizedHeader: update.FinalizedHeader,
		FinalityBranch:  update.FinalityBranch,
		SyncAggregate:   update.SyncAggregate,
		SignatureSlot:   update.SignatureSlot,
	}
}

func NewLightClientUpdateFromOptimisticUpdate(update *ethpbv2.LightClientOptimisticUpdate) *ethpbv2.LightClientUpdate {
	return &ethpbv2.LightClientUpdate{
		AttestedHeader: update.AttestedHeader,
		SyncAggregate:  update.SyncAggregate,
		SignatureSlot:  update.SignatureSlot,
	}
}
