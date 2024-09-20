package light_client_test

import (
	"testing"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	v11 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"

	lightClient "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/light-client"
	light_client "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/light-client"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

// TODO: check for attested block in tests

// NewLightClientOptimisticUpdateFromBeaconState tests

func TestLightClient_NewLightClientOptimisticUpdateFromBeaconStateAltair(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestAltair()

	update, err := lightClient.NewLightClientOptimisticUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)
}

func TestLightClient_NewLightClientOptimisticUpdateFromBeaconStateCapella(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestCapella(false)

	update, err := lightClient.NewLightClientOptimisticUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)
}

func TestLightClient_NewLightClientOptimisticUpdateFromBeaconStateDeneb(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestDeneb(false)

	update, err := lightClient.NewLightClientOptimisticUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)
}

// NewLightClientFinalityUpdateFromBeaconState tests

func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateAltair(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestAltair()

	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock, l.FinalizedBlock)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)

	finalizedBlockHeader, err := l.FinalizedBlock.Header()
	require.NoError(t, err)

	//zeroHash := params.BeaconConfig().ZeroHash[:]
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, finalizedBlockHeader.Header.Slot, updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not equal")
	require.Equal(t, finalizedBlockHeader.Header.ProposerIndex, updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.ParentRoot, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.StateRoot, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.BodyRoot, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not equal")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")

	finalityBranch, err := l.AttestedState.FinalizedRootProof(l.Ctx)
	require.NoError(t, err)
	for i, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, finalityBranch[i], leaf, "Leaf is not equal")
	}
}

func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateCapella(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestCapella(false)
	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock, l.FinalizedBlock)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)

	finalizedBlockHeader, err := l.FinalizedBlock.Header()
	require.NoError(t, err)
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, finalizedBlockHeader.Header.Slot, updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not equal")
	require.Equal(t, finalizedBlockHeader.Header.ProposerIndex, updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.ParentRoot, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.StateRoot, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.BodyRoot, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not equal")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	finalityBranch, err := l.AttestedState.FinalizedRootProof(l.Ctx)
	require.NoError(t, err)
	for i, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, finalityBranch[i], leaf, "Leaf is not equal")
	}

	// Check Execution BlockHash
	payloadInterface, err := l.FinalizedBlock.Block().Body().Execution()
	require.NoError(t, err)
	transactionsRoot, err := payloadInterface.TransactionsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		transactions, err := payloadInterface.Transactions()
		require.NoError(t, err)
		transactionsRootArray, err := ssz.TransactionsRoot(transactions)
		require.NoError(t, err)
		transactionsRoot = transactionsRootArray[:]
	} else {
		require.NoError(t, err)
	}
	withdrawalsRoot, err := payloadInterface.WithdrawalsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		withdrawals, err := payloadInterface.Withdrawals()
		require.NoError(t, err)
		withdrawalsRootArray, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		require.NoError(t, err)
		withdrawalsRoot = withdrawalsRootArray[:]
	} else {
		require.NoError(t, err)
	}
	execution := &v11.ExecutionPayloadHeaderCapella{
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
	require.DeepSSZEqual(t, execution, update.FinalizedHeader.GetHeaderCapella().Execution, "Finalized Block Execution is not equal")
}

func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateDeneb(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestDeneb(false)

	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock, l.FinalizedBlock)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)

	//zeroHash := params.BeaconConfig().ZeroHash[:]
	finalizedBlockHeader, err := l.FinalizedBlock.Header()
	require.NoError(t, err)
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, finalizedBlockHeader.Header.Slot, updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not equal")
	require.Equal(t, finalizedBlockHeader.Header.ProposerIndex, updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.ParentRoot, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.StateRoot, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.BodyRoot, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not equal")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	finalityBranch, err := l.AttestedState.FinalizedRootProof(l.Ctx)
	require.NoError(t, err)
	for i, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, finalityBranch[i], leaf, "Leaf is not equal")
	}

	// Check Execution BlockHash
	payloadInterface, err := l.FinalizedBlock.Block().Body().Execution()
	require.NoError(t, err)
	transactionsRoot, err := payloadInterface.TransactionsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		transactions, err := payloadInterface.Transactions()
		require.NoError(t, err)
		transactionsRootArray, err := ssz.TransactionsRoot(transactions)
		require.NoError(t, err)
		transactionsRoot = transactionsRootArray[:]
	} else {
		require.NoError(t, err)
	}
	withdrawalsRoot, err := payloadInterface.WithdrawalsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		withdrawals, err := payloadInterface.Withdrawals()
		require.NoError(t, err)
		withdrawalsRootArray, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		require.NoError(t, err)
		withdrawalsRoot = withdrawalsRootArray[:]
	} else {
		require.NoError(t, err)
	}
	execution := &v11.ExecutionPayloadHeaderDeneb{
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
	require.DeepSSZEqual(t, execution, update.FinalizedHeader.GetHeaderDeneb().Execution, "Finalized Block Execution is not equal")
}

func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateAltairFinalizedBlockNil(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestAltair()

	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock, nil)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)

	zeroHash := params.BeaconConfig().ZeroHash[:]
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, primitives.Slot(0), updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not zero")
	require.Equal(t, primitives.ValidatorIndex(0), updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not zero")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	for _, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, zeroHash, leaf, "Leaf is not zero")
	}
}

func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateCapellaFinalizedBlockNil(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestCapella(false)
	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock, nil)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)

	zeroHash := params.BeaconConfig().ZeroHash[:]
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, primitives.Slot(0), updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not zero")
	require.Equal(t, primitives.ValidatorIndex(0), updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not zero")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	for _, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, zeroHash, leaf, "Leaf is not zero")
	}
}

func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateDenebFinalizedBlockNil(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestDeneb(false)

	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock, nil)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)

	zeroHash := params.BeaconConfig().ZeroHash[:]
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, primitives.Slot(0), updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not zero")
	require.Equal(t, primitives.ValidatorIndex(0), updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not zero")
	require.DeepSSZEqual(t, zeroHash, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not zero")
	require.DeepSSZEqual(t, zeroHash, update.FinalizedHeader.GetHeaderDeneb().Execution.BlockHash, "Execution BlockHash is not zero")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	for _, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, zeroHash, leaf, "Leaf is not zero")
	}
}

func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateCapellaFinalizedBlockInPreviousFork(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestCapellaFinalizedBlockAltair(false)
	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock, l.FinalizedBlock)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)

	finalizedBlockHeader, err := l.FinalizedBlock.Header()
	require.NoError(t, err)
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, finalizedBlockHeader.Header.Slot, updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not equal")
	require.Equal(t, finalizedBlockHeader.Header.ProposerIndex, updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.ParentRoot, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.StateRoot, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.BodyRoot, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not equal")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	finalityBranch, err := l.AttestedState.FinalizedRootProof(l.Ctx)
	require.NoError(t, err)
	for i, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, finalityBranch[i], leaf, "Leaf is not equal")
	}

}

func TestLightClient_NewLightClientFinalityUpdateFromBeaconStateDenebFinalizedBlockInPreviousFork(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestDenebFinalizedBlockCapella(false)

	update, err := lightClient.NewLightClientFinalityUpdateFromBeaconState(l.Ctx, l.State, l.Block, l.AttestedState, l.AttestedBlock, l.FinalizedBlock)
	require.NoError(t, err)
	require.NotNil(t, update, "update is nil")

	require.Equal(t, l.Block.Block().Slot(), update.SignatureSlot, "Signature slot is not equal")

	l.CheckSyncAggregate(update.SyncAggregate)
	l.CheckAttestedHeader(update.AttestedHeader)

	//zeroHash := params.BeaconConfig().ZeroHash[:]
	finalizedBlockHeader, err := l.FinalizedBlock.Header()
	require.NoError(t, err)
	require.NotNil(t, update.FinalizedHeader, "Finalized header is nil")
	updateFinalizedHeaderBeacon, err := update.FinalizedHeader.GetBeacon()
	require.NoError(t, err)
	require.Equal(t, finalizedBlockHeader.Header.Slot, updateFinalizedHeaderBeacon.Slot, "Finalized header slot is not equal")
	require.Equal(t, finalizedBlockHeader.Header.ProposerIndex, updateFinalizedHeaderBeacon.ProposerIndex, "Finalized header proposer index is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.ParentRoot, updateFinalizedHeaderBeacon.ParentRoot, "Finalized header parent root is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.StateRoot, updateFinalizedHeaderBeacon.StateRoot, "Finalized header state root is not equal")
	require.DeepSSZEqual(t, finalizedBlockHeader.Header.BodyRoot, updateFinalizedHeaderBeacon.BodyRoot, "Finalized header body root is not equal")
	require.Equal(t, lightClient.FinalityBranchNumOfLeaves, len(update.FinalityBranch), "Invalid finality branch leaves")
	finalityBranch, err := l.AttestedState.FinalizedRootProof(l.Ctx)
	require.NoError(t, err)
	for i, leaf := range update.FinalityBranch {
		require.DeepSSZEqual(t, finalityBranch[i], leaf, "Leaf is not equal")
	}

	// Check Execution BlockHash
	payloadInterface, err := l.FinalizedBlock.Block().Body().Execution()
	require.NoError(t, err)
	transactionsRoot, err := payloadInterface.TransactionsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		transactions, err := payloadInterface.Transactions()
		require.NoError(t, err)
		transactionsRootArray, err := ssz.TransactionsRoot(transactions)
		require.NoError(t, err)
		transactionsRoot = transactionsRootArray[:]
	} else {
		require.NoError(t, err)
	}
	withdrawalsRoot, err := payloadInterface.WithdrawalsRoot()
	if errors.Is(err, consensus_types.ErrUnsupportedField) {
		withdrawals, err := payloadInterface.Withdrawals()
		require.NoError(t, err)
		withdrawalsRootArray, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		require.NoError(t, err)
		withdrawalsRoot = withdrawalsRootArray[:]
	} else {
		require.NoError(t, err)
	}
	execution := &v11.ExecutionPayloadHeaderCapella{
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
	require.DeepSSZEqual(t, execution, update.FinalizedHeader.GetHeaderCapella().Execution, "Finalized Block Execution is not equal")
}

// BlockToLightClientHeader tests

func TestLightClient_BlockToLightClientHeaderAltair(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestAltair()

	header, err := lightClient.BlockToLightClientHeaderAltair(l.Block)
	require.NoError(t, err)
	require.NotNil(t, header, "header is nil")

	parentRoot := l.Block.Block().ParentRoot()
	stateRoot := l.Block.Block().StateRoot()
	bodyRoot, err := l.Block.Block().Body().HashTreeRoot()
	require.NoError(t, err)

	require.Equal(t, l.Block.Block().Slot(), header.Beacon.Slot, "Slot is not equal")
	require.Equal(t, l.Block.Block().ProposerIndex(), header.Beacon.ProposerIndex, "Proposer index is not equal")
	require.DeepSSZEqual(t, parentRoot[:], header.Beacon.ParentRoot, "Parent root is not equal")
	require.DeepSSZEqual(t, stateRoot[:], header.Beacon.StateRoot, "State root is not equal")
	require.DeepSSZEqual(t, bodyRoot[:], header.Beacon.BodyRoot, "Body root is not equal")
}

func TestLightClient_BlockToLightClientHeaderCapella_NonBlindedBeaconBlock(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestCapella(false)

	header, err := lightClient.BlockToLightClientHeaderCapella(l.Ctx, l.Block)
	require.NoError(t, err)
	require.NotNil(t, header, "header is nil")

	parentRoot := l.Block.Block().ParentRoot()
	stateRoot := l.Block.Block().StateRoot()
	bodyRoot, err := l.Block.Block().Body().HashTreeRoot()
	require.NoError(t, err)

	payload, err := l.Block.Block().Body().Execution()
	require.NoError(t, err)

	transactionsRoot, err := light_client.ComputeTransactionsRoot(payload)
	require.NoError(t, err)

	withdrawalsRoot, err := light_client.ComputeWithdrawalsRoot(payload)
	require.NoError(t, err)

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

	executionPayloadProof, err := blocks.PayloadProof(l.Ctx, l.Block.Block())
	require.NoError(t, err)

	require.Equal(t, l.Block.Block().Slot(), header.Beacon.Slot, "Slot is not equal")
	require.Equal(t, l.Block.Block().ProposerIndex(), header.Beacon.ProposerIndex, "Proposer index is not equal")
	require.DeepSSZEqual(t, parentRoot[:], header.Beacon.ParentRoot, "Parent root is not equal")
	require.DeepSSZEqual(t, stateRoot[:], header.Beacon.StateRoot, "State root is not equal")
	require.DeepSSZEqual(t, bodyRoot[:], header.Beacon.BodyRoot, "Body root is not equal")

	require.DeepSSZEqual(t, executionHeader, header.Execution, "Execution headers are not equal")

	require.DeepSSZEqual(t, executionPayloadProof, header.ExecutionBranch, "Execution payload proofs are not equal")
}

func TestLightClient_BlockToLightClientHeaderCapella_BlindedBeaconBlock(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestCapella(true)

	header, err := lightClient.BlockToLightClientHeaderCapella(l.Ctx, l.Block)
	require.NoError(t, err)
	require.NotNil(t, header, "header is nil")

	parentRoot := l.Block.Block().ParentRoot()
	stateRoot := l.Block.Block().StateRoot()
	bodyRoot, err := l.Block.Block().Body().HashTreeRoot()
	require.NoError(t, err)

	payload, err := l.Block.Block().Body().Execution()
	require.NoError(t, err)

	transactionsRoot, err := payload.TransactionsRoot()
	require.NoError(t, err)

	withdrawalsRoot, err := payload.WithdrawalsRoot()
	require.NoError(t, err)

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

	executionPayloadProof, err := blocks.PayloadProof(l.Ctx, l.Block.Block())
	require.NoError(t, err)

	require.Equal(t, l.Block.Block().Slot(), header.Beacon.Slot, "Slot is not equal")
	require.Equal(t, l.Block.Block().ProposerIndex(), header.Beacon.ProposerIndex, "Proposer index is not equal")
	require.DeepSSZEqual(t, parentRoot[:], header.Beacon.ParentRoot, "Parent root is not equal")
	require.DeepSSZEqual(t, stateRoot[:], header.Beacon.StateRoot, "State root is not equal")
	require.DeepSSZEqual(t, bodyRoot[:], header.Beacon.BodyRoot, "Body root is not equal")

	require.DeepSSZEqual(t, executionHeader, header.Execution, "Execution headers are not equal")

	require.DeepSSZEqual(t, executionPayloadProof, header.ExecutionBranch, "Execution payload proofs are not equal")
}

func TestLightClient_BlockToLightClientHeaderDeneb_NonBlindedBeaconBlock(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestDeneb(false)

	header, err := lightClient.BlockToLightClientHeaderDeneb(l.Ctx, l.Block)
	require.NoError(t, err)
	require.NotNil(t, header, "header is nil")

	parentRoot := l.Block.Block().ParentRoot()
	stateRoot := l.Block.Block().StateRoot()
	bodyRoot, err := l.Block.Block().Body().HashTreeRoot()
	require.NoError(t, err)

	payload, err := l.Block.Block().Body().Execution()
	require.NoError(t, err)

	transactionsRoot, err := light_client.ComputeTransactionsRoot(payload)
	require.NoError(t, err)

	withdrawalsRoot, err := light_client.ComputeWithdrawalsRoot(payload)
	require.NoError(t, err)

	blobGasUsed, err := payload.BlobGasUsed()
	require.NoError(t, err)

	excessBlobGas, err := payload.ExcessBlobGas()
	require.NoError(t, err)

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

	executionPayloadProof, err := blocks.PayloadProof(l.Ctx, l.Block.Block())
	require.NoError(t, err)

	require.Equal(t, l.Block.Block().Slot(), header.Beacon.Slot, "Slot is not equal")
	require.Equal(t, l.Block.Block().ProposerIndex(), header.Beacon.ProposerIndex, "Proposer index is not equal")
	require.DeepSSZEqual(t, parentRoot[:], header.Beacon.ParentRoot, "Parent root is not equal")
	require.DeepSSZEqual(t, stateRoot[:], header.Beacon.StateRoot, "State root is not equal")
	require.DeepSSZEqual(t, bodyRoot[:], header.Beacon.BodyRoot, "Body root is not equal")

	require.DeepSSZEqual(t, executionHeader, header.Execution, "Execution headers are not equal")

	require.DeepSSZEqual(t, executionPayloadProof, header.ExecutionBranch, "Execution payload proofs are not equal")
}

func TestLightClient_BlockToLightClientHeaderDeneb_BlindedBeaconBlock(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestDeneb(true)

	header, err := lightClient.BlockToLightClientHeaderDeneb(l.Ctx, l.Block)
	require.NoError(t, err)
	require.NotNil(t, header, "header is nil")

	parentRoot := l.Block.Block().ParentRoot()
	stateRoot := l.Block.Block().StateRoot()
	bodyRoot, err := l.Block.Block().Body().HashTreeRoot()
	require.NoError(t, err)

	payload, err := l.Block.Block().Body().Execution()
	require.NoError(t, err)

	transactionsRoot, err := payload.TransactionsRoot()
	require.NoError(t, err)

	withdrawalsRoot, err := payload.WithdrawalsRoot()
	require.NoError(t, err)

	blobGasUsed, err := payload.BlobGasUsed()
	require.NoError(t, err)

	excessBlobGas, err := payload.ExcessBlobGas()
	require.NoError(t, err)

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

	executionPayloadProof, err := blocks.PayloadProof(l.Ctx, l.Block.Block())
	require.NoError(t, err)

	require.Equal(t, l.Block.Block().Slot(), header.Beacon.Slot, "Slot is not equal")
	require.Equal(t, l.Block.Block().ProposerIndex(), header.Beacon.ProposerIndex, "Proposer index is not equal")
	require.DeepSSZEqual(t, parentRoot[:], header.Beacon.ParentRoot, "Parent root is not equal")
	require.DeepSSZEqual(t, stateRoot[:], header.Beacon.StateRoot, "State root is not equal")
	require.DeepSSZEqual(t, bodyRoot[:], header.Beacon.BodyRoot, "Body root is not equal")

	require.DeepSSZEqual(t, executionHeader, header.Execution, "Execution headers are not equal")

	require.DeepSSZEqual(t, executionPayloadProof, header.ExecutionBranch, "Execution payload proofs are not equal")
}
