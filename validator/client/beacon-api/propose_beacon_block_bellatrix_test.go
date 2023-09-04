package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/mock"
	test_helpers "github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api/test-helpers"
)

func TestProposeBeaconBlock_Bellatrix(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	jsonRestHandler := mock.NewMockjsonRestHandler(ctrl)

	bellatrixBlock := generateSignedBellatrixBlock()

	genericSignedBlock := &ethpb.GenericSignedBeaconBlock{}
	genericSignedBlock.Block = bellatrixBlock

	jsonBellatrixBlock := &apimiddleware.SignedBeaconBlockBellatrixJson{
		Signature: hexutil.Encode(bellatrixBlock.Bellatrix.Signature),
		Message: &apimiddleware.BeaconBlockBellatrixJson{
			ParentRoot:    hexutil.Encode(bellatrixBlock.Bellatrix.Block.ParentRoot),
			ProposerIndex: uint64ToString(bellatrixBlock.Bellatrix.Block.ProposerIndex),
			Slot:          uint64ToString(bellatrixBlock.Bellatrix.Block.Slot),
			StateRoot:     hexutil.Encode(bellatrixBlock.Bellatrix.Block.StateRoot),
			Body: &apimiddleware.BeaconBlockBodyBellatrixJson{
				Attestations:      jsonifyAttestations(bellatrixBlock.Bellatrix.Block.Body.Attestations),
				AttesterSlashings: jsonifyAttesterSlashings(bellatrixBlock.Bellatrix.Block.Body.AttesterSlashings),
				Deposits:          jsonifyDeposits(bellatrixBlock.Bellatrix.Block.Body.Deposits),
				Eth1Data:          jsonifyEth1Data(bellatrixBlock.Bellatrix.Block.Body.Eth1Data),
				Graffiti:          hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.Graffiti),
				ProposerSlashings: jsonifyProposerSlashings(bellatrixBlock.Bellatrix.Block.Body.ProposerSlashings),
				RandaoReveal:      hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.RandaoReveal),
				VoluntaryExits:    JsonifySignedVoluntaryExits(bellatrixBlock.Bellatrix.Block.Body.VoluntaryExits),
				SyncAggregate: &apimiddleware.SyncAggregateJson{
					SyncCommitteeBits:      hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.SyncAggregate.SyncCommitteeBits),
					SyncCommitteeSignature: hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.SyncAggregate.SyncCommitteeSignature),
				},
				ExecutionPayload: &apimiddleware.ExecutionPayloadJson{
					BaseFeePerGas: bytesutil.LittleEndianBytesToBigInt(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.BaseFeePerGas).String(),
					BlockHash:     hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.BlockHash),
					BlockNumber:   uint64ToString(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.BlockNumber),
					ExtraData:     hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.ExtraData),
					FeeRecipient:  hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.FeeRecipient),
					GasLimit:      uint64ToString(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.GasLimit),
					GasUsed:       uint64ToString(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.GasUsed),
					LogsBloom:     hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.LogsBloom),
					ParentHash:    hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.ParentHash),
					PrevRandao:    hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.PrevRandao),
					ReceiptsRoot:  hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.ReceiptsRoot),
					StateRoot:     hexutil.Encode(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.StateRoot),
					TimeStamp:     uint64ToString(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.Timestamp),
					Transactions:  jsonifyTransactions(bellatrixBlock.Bellatrix.Block.Body.ExecutionPayload.Transactions),
				},
			},
		},
	}

	marshalledBlock, err := json.Marshal(jsonBellatrixBlock)
	require.NoError(t, err)

	ctx := context.Background()

	// Make sure that what we send in the POST body is the marshalled version of the protobuf block
	headers := map[string]string{"Eth-Consensus-Version": "bellatrix"}
	jsonRestHandler.EXPECT().PostRestJson(
		ctx,
		"/eth/v1/beacon/blocks",
		headers,
		bytes.NewBuffer(marshalledBlock),
		nil,
	)

	validatorClient := &beaconApiValidatorClient{jsonRestHandler: jsonRestHandler}
	proposeResponse, err := validatorClient.proposeBeaconBlock(ctx, genericSignedBlock)
	assert.NoError(t, err)
	require.NotNil(t, proposeResponse)

	expectedBlockRoot, err := bellatrixBlock.Bellatrix.Block.HashTreeRoot()
	require.NoError(t, err)

	// Make sure that the block root is set
	assert.DeepEqual(t, expectedBlockRoot[:], proposeResponse.BlockRoot)
}

func generateSignedBellatrixBlock() *ethpb.GenericSignedBeaconBlock_Bellatrix {
	return &ethpb.GenericSignedBeaconBlock_Bellatrix{
		Bellatrix: &ethpb.SignedBeaconBlockBellatrix{
			Block:     test_helpers.GenerateProtoBellatrixBeaconBlock(),
			Signature: test_helpers.FillByteSlice(96, 127),
		},
	}
}
