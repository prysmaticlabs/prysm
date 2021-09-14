package apimiddleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/encoding/bytes"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/shared/gateway"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestWrapAttestationArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitAttestationRequestJson{},
		}
		unwrappedAtts := []*attestationJson{{AggregationBits: "1010"}}
		unwrappedAttsJson, err := json.Marshal(unwrappedAtts)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedAttsJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapAttestationsArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		wrappedAtts := &submitAttestationRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedAtts))
		require.Equal(t, 1, len(wrappedAtts.Data), "wrong number of wrapped items")
		assert.Equal(t, "1010", wrappedAtts.Data[0].AggregationBits)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitAttestationRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapAttestationsArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapValidatorIndicesArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &dutiesRequestJson{},
		}
		unwrappedIndices := []string{"1", "2"}
		unwrappedIndicesJson, err := json.Marshal(unwrappedIndices)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedIndicesJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapValidatorIndicesArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		wrappedIndices := &dutiesRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedIndices))
		require.Equal(t, 2, len(wrappedIndices.Index), "wrong number of wrapped items")
		assert.Equal(t, "1", wrappedIndices.Index[0])
		assert.Equal(t, "2", wrappedIndices.Index[1])
	})
}

func TestWrapSignedAggregateAndProofArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitAggregateAndProofsRequestJson{},
		}
		unwrappedAggs := []*signedAggregateAttestationAndProofJson{{Signature: "sig"}}
		unwrappedAggsJson, err := json.Marshal(unwrappedAggs)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedAggsJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapSignedAggregateAndProofArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		wrappedAggs := &submitAggregateAndProofsRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedAggs))
		require.Equal(t, 1, len(wrappedAggs.Data), "wrong number of wrapped items")
		assert.Equal(t, "sig", wrappedAggs.Data[0].Signature)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitAggregateAndProofsRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapSignedAggregateAndProofArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapBeaconCommitteeSubscriptionsArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitBeaconCommitteeSubscriptionsRequestJson{},
		}
		unwrappedSubs := []*beaconCommitteeSubscribeJson{{
			ValidatorIndex:   "1",
			CommitteeIndex:   "1",
			CommitteesAtSlot: "1",
			Slot:             "1",
			IsAggregator:     true,
		}}
		unwrappedSubsJson, err := json.Marshal(unwrappedSubs)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedSubsJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapBeaconCommitteeSubscriptionsArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		wrappedSubs := &submitBeaconCommitteeSubscriptionsRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedSubs))
		require.Equal(t, 1, len(wrappedSubs.Data), "wrong number of wrapped items")
		assert.Equal(t, "1", wrappedSubs.Data[0].ValidatorIndex)
		assert.Equal(t, "1", wrappedSubs.Data[0].CommitteeIndex)
		assert.Equal(t, "1", wrappedSubs.Data[0].CommitteesAtSlot)
		assert.Equal(t, "1", wrappedSubs.Data[0].Slot)
		assert.Equal(t, true, wrappedSubs.Data[0].IsAggregator)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitBeaconCommitteeSubscriptionsRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapBeaconCommitteeSubscriptionsArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapSyncCommitteeSubscriptionsArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitSyncCommitteeSubscriptionRequestJson{},
		}
		unwrappedSubs := []*syncCommitteeSubscriptionJson{
			{
				ValidatorIndex:       "1",
				SyncCommitteeIndices: []string{"1", "2"},
				UntilEpoch:           "1",
			},
			{
				ValidatorIndex:       "2",
				SyncCommitteeIndices: []string{"3", "4"},
				UntilEpoch:           "2",
			},
		}
		unwrappedSubsJson, err := json.Marshal(unwrappedSubs)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedSubsJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapSyncCommitteeSubscriptionsArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		wrappedSubs := &submitSyncCommitteeSubscriptionRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedSubs))
		require.Equal(t, 2, len(wrappedSubs.Data), "wrong number of wrapped items")
		assert.Equal(t, "1", wrappedSubs.Data[0].ValidatorIndex)
		require.Equal(t, 2, len(wrappedSubs.Data[0].SyncCommitteeIndices), "wrong number of committee indices")
		assert.Equal(t, "1", wrappedSubs.Data[0].SyncCommitteeIndices[0])
		assert.Equal(t, "2", wrappedSubs.Data[0].SyncCommitteeIndices[1])
		assert.Equal(t, "1", wrappedSubs.Data[0].UntilEpoch)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitSyncCommitteeSubscriptionRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapSyncCommitteeSubscriptionsArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapSyncCommitteeSignaturesArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitSyncCommitteeSignaturesRequestJson{},
		}
		unwrappedSigs := []*syncCommitteeMessageJson{{
			Slot:            "1",
			BeaconBlockRoot: "root",
			ValidatorIndex:  "1",
			Signature:       "sig",
		}}
		unwrappedSigsJson, err := json.Marshal(unwrappedSigs)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedSigsJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapSyncCommitteeSignaturesArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		wrappedSigs := &submitSyncCommitteeSignaturesRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedSigs))
		require.Equal(t, 1, len(wrappedSigs.Data), "wrong number of wrapped items")
		assert.Equal(t, "1", wrappedSigs.Data[0].Slot)
		assert.Equal(t, "root", wrappedSigs.Data[0].BeaconBlockRoot)
		assert.Equal(t, "1", wrappedSigs.Data[0].ValidatorIndex)
		assert.Equal(t, "sig", wrappedSigs.Data[0].Signature)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitSyncCommitteeSignaturesRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapSyncCommitteeSignaturesArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapSignedContributionAndProofsArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitContributionAndProofsRequestJson{},
		}
		unwrapped := []*signedContributionAndProofJson{
			{
				Message: &contributionAndProofJson{
					AggregatorIndex: "1",
					Contribution: &syncCommitteeContributionJson{
						Slot:              "1",
						BeaconBlockRoot:   "root",
						SubcommitteeIndex: "1",
						AggregationBits:   "bits",
						Signature:         "sig",
					},
					SelectionProof: "proof",
				},
				Signature: "sig",
			},
			{
				Message:   &contributionAndProofJson{},
				Signature: "sig",
			},
		}
		unwrappedJson, err := json.Marshal(unwrapped)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapSignedContributionAndProofsArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		wrapped := &submitContributionAndProofsRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrapped))
		require.Equal(t, 2, len(wrapped.Data), "wrong number of wrapped items")
		assert.Equal(t, "sig", wrapped.Data[0].Signature)
		require.NotNil(t, wrapped.Data[0].Message)
		msg := wrapped.Data[0].Message
		assert.Equal(t, "1", msg.AggregatorIndex)
		assert.Equal(t, "proof", msg.SelectionProof)
		require.NotNil(t, msg.Contribution)
		assert.Equal(t, "1", msg.Contribution.Slot)
		assert.Equal(t, "root", msg.Contribution.BeaconBlockRoot)
		assert.Equal(t, "1", msg.Contribution.SubcommitteeIndex)
		assert.Equal(t, "bits", msg.Contribution.AggregationBits)
		assert.Equal(t, "sig", msg.Contribution.Signature)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := gateway.Endpoint{
			PostRequest: &submitContributionAndProofsRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		errJson := wrapSignedContributionAndProofsArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestPrepareGraffiti(t *testing.T) {
	endpoint := gateway.Endpoint{
		PostRequest: &beaconBlockContainerJson{
			Message: &beaconBlockJson{
				Body: &beaconBlockBodyJson{},
			},
		},
	}

	t.Run("32_bytes", func(t *testing.T) {
		endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti = string(bytes.PadTo([]byte("foo"), 32))

		prepareGraffiti(endpoint, nil, nil)
		assert.Equal(
			t,
			"0x666f6f0000000000000000000000000000000000000000000000000000000000",
			endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti,
		)
	})

	t.Run("less_than_32_bytes", func(t *testing.T) {
		endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti = "foo"

		prepareGraffiti(endpoint, nil, nil)
		assert.Equal(
			t,
			"0x666f6f0000000000000000000000000000000000000000000000000000000000",
			endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti,
		)
	})

	t.Run("more_than_32_bytes", func(t *testing.T) {
		endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti = string(bytes.PadTo([]byte("foo"), 33))

		prepareGraffiti(endpoint, nil, nil)
		assert.Equal(
			t,
			"0x666f6f0000000000000000000000000000000000000000000000000000000000",
			endpoint.PostRequest.(*beaconBlockContainerJson).Message.Body.Graffiti,
		)
	})
}

func TestPrepareValidatorAggregates(t *testing.T) {
	body := &tempSyncCommitteesResponseJson{
		Data: &tempSyncCommitteeValidatorsJson{
			Validators: []string{"1", "2"},
			ValidatorAggregates: []*tempSyncSubcommitteeValidatorsJson{
				{
					Validators: []string{"3", "4"},
				},
				{
					Validators: []string{"5"},
				},
			},
		},
	}
	bodyJson, err := json.Marshal(body)
	require.NoError(t, err)

	container := &syncCommitteesResponseJson{}
	handled, errJson := prepareValidatorAggregates(bodyJson, container)
	require.Equal(t, nil, errJson)
	require.Equal(t, true, handled)
	assert.DeepEqual(t, []string{"1", "2"}, container.Data.Validators)
	require.DeepEqual(t, [][]string{{"3", "4"}, {"5"}}, container.Data.ValidatorAggregates)
}

func TestSerializeV2Block(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		response := &blockV2ResponseJson{
			Version: ethpbv2.Version_PHASE0.String(),
			Data: &beaconBlockContainerV2Json{
				Phase0Block: &beaconBlockJson{},
				AltairBlock: nil,
				Signature:   "sig",
			},
		}
		ok, j, errJson := serializeV2Block(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, true, ok)
		require.NotNil(t, j)
		require.NoError(t, json.Unmarshal(j, &phase0BlockResponseJson{}))
	})

	t.Run("Altair", func(t *testing.T) {
		response := &blockV2ResponseJson{
			Version: ethpbv2.Version_ALTAIR.String(),
			Data: &beaconBlockContainerV2Json{
				Phase0Block: nil,
				AltairBlock: &beaconBlockAltairJson{},
				Signature:   "sig",
			},
		}
		ok, j, errJson := serializeV2Block(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, true, ok)
		require.NotNil(t, j)
		require.NoError(t, json.Unmarshal(j, &altairBlockResponseJson{}))
	})

	t.Run("incorrect response type", func(t *testing.T) {
		response := &types.Empty{}
		ok, j, errJson := serializeV2Block(response)
		require.Equal(t, false, ok)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "container is not of the correct type"))
	})
}

func TestSerializeV2State(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		response := &beaconStateV2ResponseJson{
			Version: ethpbv2.Version_PHASE0.String(),
			Data: &beaconStateContainerV2Json{
				Phase0State: &beaconStateJson{},
				AltairState: nil,
			},
		}
		ok, j, errJson := serializeV2State(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, true, ok)
		require.NotNil(t, j)
		require.NoError(t, json.Unmarshal(j, &phase0StateResponseJson{}))
	})

	t.Run("Altair", func(t *testing.T) {
		response := &beaconStateV2ResponseJson{
			Version: ethpbv2.Version_ALTAIR.String(),
			Data: &beaconStateContainerV2Json{
				Phase0State: nil,
				AltairState: &beaconStateV2Json{},
			},
		}
		ok, j, errJson := serializeV2State(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, true, ok)
		require.NotNil(t, j)
		require.NoError(t, json.Unmarshal(j, &altairStateResponseJson{}))
	})

	t.Run("incorrect response type", func(t *testing.T) {
		ok, j, errJson := serializeV2State(&types.Empty{})
		require.Equal(t, false, ok)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "container is not of the correct type"))
	})
}
