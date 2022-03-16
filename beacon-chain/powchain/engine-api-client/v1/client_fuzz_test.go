//go:build go1.18
// +build go1.18

package v1_test

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/beacon"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"math"
	"math/big"
	"testing"
)

func FuzzForkChoiceResponse(f *testing.F) {
	valHash := common.Hash([32]byte{0xFF, 0x01})
	payloadID := beacon.PayloadID([8]byte{0x01, 0xFF, 0xAA, 0x00, 0xEE, 0xFE, 0x00, 0x00})
	valErr := "asjajshjahsaj"
	seed := &beacon.ForkChoiceResponse{
		PayloadStatus: beacon.PayloadStatusV1{
			Status:          "INVALID_TERMINAL_BLOCK",
			LatestValidHash: &valHash,
			ValidationError: &valErr,
		},
		PayloadID: &payloadID,
	}
	output, err := json.Marshal(seed)
	assert.NoError(f, err)
	f.Add(output)
	f.Fuzz(func(t *testing.T, jsonBlob []byte) {
		gethResp := &beacon.ForkChoiceResponse{}
		prysmResp := &v1.ForkchoiceUpdatedResponse{}
		gethErr := json.Unmarshal(jsonBlob, gethResp)
		prysmErr := json.Unmarshal(jsonBlob, prysmResp)
		assert.Equal(t, gethErr != nil, prysmErr != nil, fmt.Sprintf("geth and prysm unmarshaller return inconsistent errors. %v and %v", gethErr, prysmErr))
		// Nothing to marshal if we have an error.
		if gethErr != nil {
			return
		}
		gethBlob, gethErr := json.Marshal(gethResp)
		prysmBlob, prysmErr := json.Marshal(prysmResp)
		assert.Equal(t, gethErr != nil, prysmErr != nil, "geth and prysm unmarshaller return inconsistent errors")
		newGethResp := &beacon.ForkChoiceResponse{}
		newGethErr := json.Unmarshal(prysmBlob, newGethResp)
		assert.NoError(t, newGethErr)
		if newGethResp.PayloadStatus.Status == "UNKNOWN" {
			return
		}

		newGethResp2 := &beacon.ForkChoiceResponse{}
		newGethErr = json.Unmarshal(gethBlob, newGethResp2)
		assert.NoError(t, newGethErr)

		assert.DeepEqual(t, newGethResp.PayloadID, newGethResp2.PayloadID)
		assert.DeepEqual(t, newGethResp.PayloadStatus.Status, newGethResp2.PayloadStatus.Status)
		assert.DeepEqual(t, newGethResp.PayloadStatus.LatestValidHash, newGethResp2.PayloadStatus.LatestValidHash)
		isNilOrEmpty := newGethResp.PayloadStatus.ValidationError == nil || (*newGethResp.PayloadStatus.ValidationError == "")
		isNilOrEmpty2 := newGethResp2.PayloadStatus.ValidationError == nil || (*newGethResp2.PayloadStatus.ValidationError == "")
		assert.DeepEqual(t, isNilOrEmpty, isNilOrEmpty2)
		if !isNilOrEmpty {
			assert.DeepEqual(t, *newGethResp.PayloadStatus.ValidationError, *newGethResp2.PayloadStatus.ValidationError)
		}
	})
}

func FuzzExecutionPayload(f *testing.F) {
	logsBloom := [256]byte{'j', 'u', 'n', 'k'}
	execData := &beacon.ExecutableDataV1{
		ParentHash:    common.Hash([32]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01}),
		FeeRecipient:  common.Address([20]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF}),
		StateRoot:     common.Hash([32]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01}),
		ReceiptsRoot:  common.Hash([32]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01}),
		LogsBloom:     logsBloom[:],
		Random:        common.Hash([32]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01}),
		Number:        math.MaxUint64,
		GasLimit:      math.MaxUint64,
		GasUsed:       math.MaxUint64,
		Timestamp:     100,
		ExtraData:     nil,
		BaseFeePerGas: big.NewInt(math.MaxInt),
		BlockHash:     common.Hash([32]byte{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01}),
		Transactions:  [][]byte{{0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01}, {0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01}, {0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01}, {0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01, 0xFF, 0x01}},
	}
	output, err := json.Marshal(execData)
	assert.NoError(f, err)
	f.Add(output)
	f.Fuzz(func(t *testing.T, jsonBlob []byte) {
		gethResp := &beacon.ExecutableDataV1{}
		prysmResp := &pb.ExecutionPayload{}
		gethErr := json.Unmarshal(jsonBlob, gethResp)
		prysmErr := json.Unmarshal(jsonBlob, prysmResp)
		assert.Equal(t, gethErr != nil, prysmErr != nil, fmt.Sprintf("geth and prysm unmarshaller return inconsistent errors. %v and %v", gethErr, prysmErr))
		// Nothing to marshal if we have an error.
		if gethErr != nil {
			return
		}
		gethBlob, gethErr := json.Marshal(gethResp)
		prysmBlob, prysmErr := json.Marshal(prysmResp)
		assert.Equal(t, gethErr != nil, prysmErr != nil, "geth and prysm unmarshaller return inconsistent errors")
		newGethResp := &beacon.ExecutableDataV1{}
		newGethErr := json.Unmarshal(prysmBlob, newGethResp)
		assert.NoError(t, newGethErr)
		newGethResp2 := &beacon.ExecutableDataV1{}
		newGethErr = json.Unmarshal(gethBlob, newGethResp2)
		assert.NoError(t, newGethErr)

		assert.DeepEqual(t, newGethResp, newGethResp2)
	})
}
