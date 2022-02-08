package main

import (
	"context"
	"encoding/json"
	"testing"

	v1 "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	"github.com/prysmaticlabs/prysm/io/file"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestEngineAPICompliance(t *testing.T) {
	ctx := context.Background()
	client, err := v1.New(ctx, "http://localhost:8550")
	require.NoError(t, err)

	// First, prepare a payload for forkchoice updates.
	type args struct {
		ForkchoiceState   *enginev1.ForkchoiceState   `json:"forkchoiceState"`
		PayloadAttributes *enginev1.PayloadAttributes `json:"payloadAttributes"`
	}
	req := args{}
	readDataFile("forkchoice_updated_request", &req)

	// engine_forkchoiceUpdatedV1.
	resp, err := client.ForkchoiceUpdated(ctx, req.ForkchoiceState, req.PayloadAttributes)
	require.NoError(t, err)
	want := &v1.ForkchoiceUpdatedResponse{}
	readDataFile("forkchoice_updated_response", &want)
	require.DeepEqual(t, want, resp)

	// This will create a payload under payload id 0xa247243752eb10b4.
	// All subsequent calls with the same parameters should create a
	// payload under the same id.

	// engine_getPayloadV1.
	var getPayloadReq enginev1.PayloadIDBytes
	require.NoError(t, json.Unmarshal([]byte("\"0xa247243752eb10b4\""), &getPayloadReq))
	getPayloadResp, err := client.GetPayload(ctx, getPayloadReq)
	require.NoError(t, err)

	wantGetPayloadResp := &enginev1.ExecutionPayload{}
	readDataFile("get_payload_response", &wantGetPayloadResp)
	require.DeepEqual(t, wantGetPayloadResp, getPayloadResp)

	// engine_newPayloadV1.
	var newPayloadReq *enginev1.ExecutionPayload
	readDataFile("new_payload_request", &newPayloadReq)
	newPayloadResp, err := client.NewPayload(ctx, newPayloadReq)
	require.NoError(t, err)

	wantNewPayloadResp := &enginev1.PayloadStatus{}
	readDataFile("new_payload_response", &wantNewPayloadResp)
	require.DeepEqual(t, wantNewPayloadResp, newPayloadResp)
}

func readDataFile(fileName string, target interface{}) {
	enc, err := file.ReadFileAsBytes("data/" + fileName + ".json")
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(enc, target); err != nil {
		panic(err)
	}
}
