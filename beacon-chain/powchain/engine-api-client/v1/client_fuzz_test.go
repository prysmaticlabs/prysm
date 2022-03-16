//go:build go1.18
// +build go1.18

package v1_test

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/beacon"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	"github.com/prysmaticlabs/prysm/testing/assert"
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
		assert.Equal(t, gethErr != nil, prysmErr != nil, "geth and prysm unmarshaller return inconsistent errors")
		// Nothing to marshal if we have an error.
		if gethErr != nil {
			return
		}
		_, gethErr = json.Marshal(gethResp)
		prysmBlob, prysmErr := json.Marshal(prysmResp)
		assert.Equal(t, gethErr != nil, prysmErr != nil, "geth and prysm unmarshaller return inconsistent errors")
		newGethResp := &beacon.ForkChoiceResponse{}
		newGethErr := json.Unmarshal(prysmBlob, newGethResp)
		assert.NoError(t, newGethErr)
		assert.DeepEqual(t, gethResp, newGethResp)
	})
}
