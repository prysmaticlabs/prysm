//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestGetGenesis_ValidAttestation(t *testing.T) {
	expectedSlot := uint64(5)
	expectedCommitteeIndex := uint64(6)
	expectedBeaconBlockRoot := "0x0636045df9bdda3ab96592cf5389032c8ec3977f911e2b53509b348dfe164d4d"
	expectedSourceEpoch := uint64(7)
	expectedSourceRoot := "0xd4bcbdefc8156e85247681086e8050e5d2d5d1bf076a25f6decd99250f3a378d"
	expectedTargetEpoch := uint64(8)
	expectedTargetRoot := "0x246590e8e4c2a9bd13cc776ecc7025bc432219f076e80b27267b8fa0456dc821"

	// Mock the /eth/v1/beacon/blocks/{slot}/attestations endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/blocks/%d/attestations", expectedSlot), createAttestationsHandler(&rpcmiddleware.BlockAttestationsResponseJson{
		Data: []*rpcmiddleware.AttestationJson{
			{
				Data: &rpcmiddleware.AttestationDataJson{
					Slot:            "5",
					CommitteeIndex:  "2",
					BeaconBlockRoot: "0x6690662ef388dde3a4f8686c2847494414fe037640b105525d7f759f951c03c9",
					Source: &rpcmiddleware.CheckpointJson{
						Epoch: "3",
						Root:  "0x9365a8acecfd885b919ef94c3bacd0a2c8a456c7eb079360fe9a3295e2dfc846",
					},
					Target: &rpcmiddleware.CheckpointJson{
						Epoch: "4",
						Root:  "0x1af95256f4d3f8655dd4731b2bb8c61e6d31417af6f8967931776ea08c72d873",
					},
				},
			},
			{
				Data: &rpcmiddleware.AttestationDataJson{
					Slot:            strconv.FormatUint(expectedSlot, 10),
					CommitteeIndex:  strconv.FormatUint(expectedCommitteeIndex, 10),
					BeaconBlockRoot: expectedBeaconBlockRoot,
					Source: &rpcmiddleware.CheckpointJson{
						Epoch: strconv.FormatUint(expectedSourceEpoch, 10),
						Root:  expectedSourceRoot,
					},
					Target: &rpcmiddleware.CheckpointJson{
						Epoch: strconv.FormatUint(expectedTargetEpoch, 10),
						Root:  expectedTargetRoot,
					},
				},
			},
		},
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	resp, err := validatorClient.getAttestationData(&eth.AttestationDataRequest{
		Slot:           types.Slot(expectedSlot),
		CommitteeIndex: types.CommitteeIndex(expectedCommitteeIndex),
	})
	assert.NoError(t, err)

	require.NotNil(t, resp)
	assert.Equal(t, expectedBeaconBlockRoot, hexutil.Encode(resp.BeaconBlockRoot))
	assert.Equal(t, expectedCommitteeIndex, uint64(resp.CommitteeIndex))
	assert.Equal(t, expectedSlot, uint64(resp.Slot))

	require.NotNil(t, resp.Source)
	assert.Equal(t, expectedSourceEpoch, uint64(resp.Source.Epoch))
	assert.Equal(t, expectedSourceRoot, hexutil.Encode(resp.Source.Root))

	require.NotNil(t, resp.Target)
	assert.Equal(t, expectedTargetEpoch, uint64(resp.Target.Epoch))
	assert.Equal(t, expectedTargetRoot, hexutil.Encode(resp.Target.Root))
}

func TestGetGenesis_MissingAttestation(t *testing.T) {
	// Mock the /eth/v1/beacon/blocks/{slot}/attestations endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/eth/v1/beacon/blocks/1/attestations"), createAttestationsHandler(&rpcmiddleware.BlockAttestationsResponseJson{
		Data: []*rpcmiddleware.AttestationJson{},
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	validatorClient := &beaconApiValidatorClient{url: server.URL, httpClient: http.Client{Timeout: time.Second * 5}}
	_, err := validatorClient.getAttestationData(&eth.AttestationDataRequest{
		Slot:           1,
		CommitteeIndex: 2,
	})
	assert.ErrorContains(t, "attestation data not found for slot `1` and committee index `2`", err)
}

func createAttestationsHandler(data *rpcmiddleware.BlockAttestationsResponseJson) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		marshalledResponse, err := json.Marshal(data)
		if err != nil {
			panic(err)
		}

		_, err = w.Write(marshalledResponse)
		if err != nil {
			panic(err)
		}
	})
}
