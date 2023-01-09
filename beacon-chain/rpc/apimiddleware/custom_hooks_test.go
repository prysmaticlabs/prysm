package apimiddleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func TestWrapAttestationArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitAttestationRequestJson{},
		}
		unwrappedAtts := []*AttestationJson{{AggregationBits: "1010"}}
		unwrappedAttsJson, err := json.Marshal(unwrappedAtts)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedAttsJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapAttestationsArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		wrappedAtts := &SubmitAttestationRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedAtts))
		require.Equal(t, 1, len(wrappedAtts.Data), "wrong number of wrapped items")
		assert.Equal(t, "1010", wrappedAtts.Data[0].AggregationBits)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitAttestationRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapAttestationsArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(false), runDefault)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapValidatorIndicesArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &ValidatorIndicesJson{},
		}
		unwrappedIndices := []string{"1", "2"}
		unwrappedIndicesJson, err := json.Marshal(unwrappedIndices)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedIndicesJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapValidatorIndicesArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		wrappedIndices := &ValidatorIndicesJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedIndices))
		require.Equal(t, 2, len(wrappedIndices.Index), "wrong number of wrapped items")
		assert.Equal(t, "1", wrappedIndices.Index[0])
		assert.Equal(t, "2", wrappedIndices.Index[1])
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &ValidatorIndicesJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapValidatorIndicesArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(false), runDefault)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapBLSChangesArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitBLSToExecutionChangesRequest{},
		}
		unwrappedChanges := []*SignedBLSToExecutionChangeJson{{Signature: "sig"}}
		unwrappedChangesJson, err := json.Marshal(unwrappedChanges)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedChangesJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapBLSChangesArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		wrappedChanges := &SubmitBLSToExecutionChangesRequest{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedChanges))
		require.Equal(t, 1, len(wrappedChanges.Changes), "wrong number of wrapped items")
		assert.Equal(t, "sig", wrappedChanges.Changes[0].Signature)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitBLSToExecutionChangesRequest{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapBLSChangesArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(false), runDefault)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapSignedAggregateAndProofArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitAggregateAndProofsRequestJson{},
		}
		unwrappedAggs := []*SignedAggregateAttestationAndProofJson{{Signature: "sig"}}
		unwrappedAggsJson, err := json.Marshal(unwrappedAggs)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedAggsJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapSignedAggregateAndProofArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		wrappedAggs := &SubmitAggregateAndProofsRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedAggs))
		require.Equal(t, 1, len(wrappedAggs.Data), "wrong number of wrapped items")
		assert.Equal(t, "sig", wrappedAggs.Data[0].Signature)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitAggregateAndProofsRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapSignedAggregateAndProofArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(false), runDefault)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapBeaconCommitteeSubscriptionsArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitBeaconCommitteeSubscriptionsRequestJson{},
		}
		unwrappedSubs := []*BeaconCommitteeSubscribeJson{{
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

		runDefault, errJson := wrapBeaconCommitteeSubscriptionsArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		wrappedSubs := &SubmitBeaconCommitteeSubscriptionsRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedSubs))
		require.Equal(t, 1, len(wrappedSubs.Data), "wrong number of wrapped items")
		assert.Equal(t, "1", wrappedSubs.Data[0].ValidatorIndex)
		assert.Equal(t, "1", wrappedSubs.Data[0].CommitteeIndex)
		assert.Equal(t, "1", wrappedSubs.Data[0].CommitteesAtSlot)
		assert.Equal(t, "1", wrappedSubs.Data[0].Slot)
		assert.Equal(t, true, wrappedSubs.Data[0].IsAggregator)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitBeaconCommitteeSubscriptionsRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapBeaconCommitteeSubscriptionsArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(false), runDefault)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapSyncCommitteeSubscriptionsArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitSyncCommitteeSubscriptionRequestJson{},
		}
		unwrappedSubs := []*SyncCommitteeSubscriptionJson{
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

		runDefault, errJson := wrapSyncCommitteeSubscriptionsArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		wrappedSubs := &SubmitSyncCommitteeSubscriptionRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedSubs))
		require.Equal(t, 2, len(wrappedSubs.Data), "wrong number of wrapped items")
		assert.Equal(t, "1", wrappedSubs.Data[0].ValidatorIndex)
		require.Equal(t, 2, len(wrappedSubs.Data[0].SyncCommitteeIndices), "wrong number of committee indices")
		assert.Equal(t, "1", wrappedSubs.Data[0].SyncCommitteeIndices[0])
		assert.Equal(t, "2", wrappedSubs.Data[0].SyncCommitteeIndices[1])
		assert.Equal(t, "1", wrappedSubs.Data[0].UntilEpoch)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitSyncCommitteeSubscriptionRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapSyncCommitteeSubscriptionsArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(false), runDefault)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapSyncCommitteeSignaturesArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitSyncCommitteeSignaturesRequestJson{},
		}
		unwrappedSigs := []*SyncCommitteeMessageJson{{
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

		runDefault, errJson := wrapSyncCommitteeSignaturesArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		wrappedSigs := &SubmitSyncCommitteeSignaturesRequestJson{}
		require.NoError(t, json.NewDecoder(request.Body).Decode(wrappedSigs))
		require.Equal(t, 1, len(wrappedSigs.Data), "wrong number of wrapped items")
		assert.Equal(t, "1", wrappedSigs.Data[0].Slot)
		assert.Equal(t, "root", wrappedSigs.Data[0].BeaconBlockRoot)
		assert.Equal(t, "1", wrappedSigs.Data[0].ValidatorIndex)
		assert.Equal(t, "sig", wrappedSigs.Data[0].Signature)
	})

	t.Run("invalid_body", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitSyncCommitteeSignaturesRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapSyncCommitteeSignaturesArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(false), runDefault)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestWrapSignedContributionAndProofsArray(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitContributionAndProofsRequestJson{},
		}
		unwrapped := []*SignedContributionAndProofJson{
			{
				Message: &ContributionAndProofJson{
					AggregatorIndex: "1",
					Contribution: &SyncCommitteeContributionJson{
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
				Message:   &ContributionAndProofJson{},
				Signature: "sig",
			},
		}
		unwrappedJson, err := json.Marshal(unwrapped)
		require.NoError(t, err)

		var body bytes.Buffer
		_, err = body.Write(unwrappedJson)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapSignedContributionAndProofsArray(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		wrapped := &SubmitContributionAndProofsRequestJson{}
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
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SubmitContributionAndProofsRequestJson{},
		}
		var body bytes.Buffer
		_, err := body.Write([]byte("invalid"))
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)

		runDefault, errJson := wrapSignedContributionAndProofsArray(endpoint, nil, request)
		require.Equal(t, false, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(false), runDefault)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "could not decode body"))
		assert.Equal(t, http.StatusInternalServerError, errJson.StatusCode())
	})
}

func TestSetInitialPublishBlockPostRequest(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.BellatrixForkEpoch = params.BeaconConfig().AltairForkEpoch + 1
	params.OverrideBeaconConfig(cfg)

	endpoint := &apimiddleware.Endpoint{}
	s := struct {
		Message struct {
			Slot string
		}
	}{}
	t.Run("Phase 0", func(t *testing.T) {
		s.Message = struct{ Slot string }{Slot: "0"}
		j, err := json.Marshal(s)
		require.NoError(t, err)
		var body bytes.Buffer
		_, err = body.Write(j)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		runDefault, errJson := setInitialPublishBlockPostRequest(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockContainerJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
	t.Run("Altair", func(t *testing.T) {
		slot, err := slots.EpochStart(params.BeaconConfig().AltairForkEpoch)
		require.NoError(t, err)
		s.Message = struct{ Slot string }{Slot: strconv.FormatUint(uint64(slot), 10)}
		j, err := json.Marshal(s)
		require.NoError(t, err)
		var body bytes.Buffer
		_, err = body.Write(j)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		runDefault, errJson := setInitialPublishBlockPostRequest(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockAltairContainerJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
	t.Run("Bellatrix", func(t *testing.T) {
		slot, err := slots.EpochStart(params.BeaconConfig().BellatrixForkEpoch)
		require.NoError(t, err)
		s.Message = struct{ Slot string }{Slot: strconv.FormatUint(uint64(slot), 10)}
		j, err := json.Marshal(s)
		require.NoError(t, err)
		var body bytes.Buffer
		_, err = body.Write(j)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		runDefault, errJson := setInitialPublishBlockPostRequest(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockBellatrixContainerJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
	t.Run("Capella", func(t *testing.T) {
		params.SetupTestConfigCleanup(t)
		cfg := params.BeaconConfig()
		cfg.CapellaForkEpoch = cfg.BellatrixForkEpoch.Add(2)
		params.OverrideBeaconConfig(cfg)

		slot, err := slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
		require.NoError(t, err)
		s.Message = struct{ Slot string }{Slot: strconv.FormatUint(uint64(slot), 10)}
		j, err := json.Marshal(s)
		require.NoError(t, err)
		var body bytes.Buffer
		_, err = body.Write(j)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		runDefault, errJson := setInitialPublishBlockPostRequest(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockCapellaContainerJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
}

func TestPreparePublishedBlock(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SignedBeaconBlockContainerJson{
				Message: &BeaconBlockJson{
					Body: &BeaconBlockBodyJson{},
				},
			},
		}
		errJson := preparePublishedBlock(endpoint, nil, nil)
		require.Equal(t, true, errJson == nil)
		_, ok := endpoint.PostRequest.(*phase0PublishBlockRequestJson)
		assert.Equal(t, true, ok)
	})

	t.Run("Altair", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SignedBeaconBlockAltairContainerJson{
				Message: &BeaconBlockAltairJson{
					Body: &BeaconBlockBodyAltairJson{},
				},
			},
		}
		errJson := preparePublishedBlock(endpoint, nil, nil)
		require.Equal(t, true, errJson == nil)
		_, ok := endpoint.PostRequest.(*altairPublishBlockRequestJson)
		assert.Equal(t, true, ok)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SignedBeaconBlockBellatrixContainerJson{
				Message: &BeaconBlockBellatrixJson{
					Body: &BeaconBlockBodyBellatrixJson{},
				},
			},
		}
		errJson := preparePublishedBlock(endpoint, nil, nil)
		require.Equal(t, true, errJson == nil)
		_, ok := endpoint.PostRequest.(*bellatrixPublishBlockRequestJson)
		assert.Equal(t, true, ok)
	})

	t.Run("unsupported block type", func(t *testing.T) {
		errJson := preparePublishedBlock(&apimiddleware.Endpoint{}, nil, nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "unsupported block type"))
	})
}

func TestSetInitialPublishBlindedBlockPostRequest(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.BellatrixForkEpoch = params.BeaconConfig().AltairForkEpoch + 1
	params.OverrideBeaconConfig(cfg)

	endpoint := &apimiddleware.Endpoint{}
	s := struct {
		Message struct {
			Slot string
		}
	}{}
	t.Run("Phase 0", func(t *testing.T) {
		s.Message = struct{ Slot string }{Slot: "0"}
		j, err := json.Marshal(s)
		require.NoError(t, err)
		var body bytes.Buffer
		_, err = body.Write(j)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		runDefault, errJson := setInitialPublishBlindedBlockPostRequest(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockContainerJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
	t.Run("Altair", func(t *testing.T) {
		slot, err := slots.EpochStart(params.BeaconConfig().AltairForkEpoch)
		require.NoError(t, err)
		s.Message = struct{ Slot string }{Slot: strconv.FormatUint(uint64(slot), 10)}
		j, err := json.Marshal(s)
		require.NoError(t, err)
		var body bytes.Buffer
		_, err = body.Write(j)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		runDefault, errJson := setInitialPublishBlindedBlockPostRequest(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockAltairContainerJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
	t.Run("Bellatrix", func(t *testing.T) {
		slot, err := slots.EpochStart(params.BeaconConfig().BellatrixForkEpoch)
		require.NoError(t, err)
		s.Message = struct{ Slot string }{Slot: strconv.FormatUint(uint64(slot), 10)}
		j, err := json.Marshal(s)
		require.NoError(t, err)
		var body bytes.Buffer
		_, err = body.Write(j)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		runDefault, errJson := setInitialPublishBlindedBlockPostRequest(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		assert.Equal(t, reflect.TypeOf(SignedBlindedBeaconBlockBellatrixContainerJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
}

func TestPreparePublishedBlindedBlock(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SignedBeaconBlockContainerJson{
				Message: &BeaconBlockJson{
					Body: &BeaconBlockBodyJson{},
				},
			},
		}
		errJson := preparePublishedBlindedBlock(endpoint, nil, nil)
		require.Equal(t, true, errJson == nil)
		_, ok := endpoint.PostRequest.(*phase0PublishBlockRequestJson)
		assert.Equal(t, true, ok)
	})

	t.Run("Altair", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SignedBeaconBlockAltairContainerJson{
				Message: &BeaconBlockAltairJson{
					Body: &BeaconBlockBodyAltairJson{},
				},
			},
		}
		errJson := preparePublishedBlindedBlock(endpoint, nil, nil)
		require.Equal(t, true, errJson == nil)
		_, ok := endpoint.PostRequest.(*altairPublishBlockRequestJson)
		assert.Equal(t, true, ok)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SignedBlindedBeaconBlockBellatrixContainerJson{
				Message: &BlindedBeaconBlockBellatrixJson{
					Body: &BlindedBeaconBlockBodyBellatrixJson{},
				},
			},
		}
		errJson := preparePublishedBlindedBlock(endpoint, nil, nil)
		require.Equal(t, true, errJson == nil)
		_, ok := endpoint.PostRequest.(*bellatrixPublishBlindedBlockRequestJson)
		assert.Equal(t, true, ok)
	})

	t.Run("unsupported block type", func(t *testing.T) {
		errJson := preparePublishedBlock(&apimiddleware.Endpoint{}, nil, nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "unsupported block type"))
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

	container := &SyncCommitteesResponseJson{}
	runDefault, errJson := prepareValidatorAggregates(bodyJson, container)
	require.Equal(t, nil, errJson)
	require.Equal(t, apimiddleware.RunDefault(false), runDefault)
	assert.DeepEqual(t, []string{"1", "2"}, container.Data.Validators)
	require.DeepEqual(t, [][]string{{"3", "4"}, {"5"}}, container.Data.ValidatorAggregates)
}

func TestSerializeV2Block(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		response := &BlockV2ResponseJson{
			Version: ethpbv2.Version_PHASE0.String(),
			Data: &SignedBeaconBlockContainerV2Json{
				Phase0Block: &BeaconBlockJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyJson{},
				},
				Signature: "sig",
			},
			ExecutionOptimistic: true,
		}
		runDefault, j, errJson := serializeV2Block(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &phase0BlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data.Message)
		beaconBlock := resp.Data.Message
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		assert.NotNil(t, beaconBlock.Body)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})

	t.Run("Altair", func(t *testing.T) {
		response := &BlockV2ResponseJson{
			Version: ethpbv2.Version_ALTAIR.String(),
			Data: &SignedBeaconBlockContainerV2Json{
				AltairBlock: &BeaconBlockAltairJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyAltairJson{},
				},
				Signature: "sig",
			},
			ExecutionOptimistic: true,
		}
		runDefault, j, errJson := serializeV2Block(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &altairBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data.Message)
		beaconBlock := resp.Data.Message
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		assert.NotNil(t, beaconBlock.Body)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		response := &BlockV2ResponseJson{
			Version: ethpbv2.Version_BELLATRIX.String(),
			Data: &SignedBeaconBlockContainerV2Json{
				BellatrixBlock: &BeaconBlockBellatrixJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyBellatrixJson{},
				},
				Signature: "sig",
			},
			ExecutionOptimistic: true,
		}
		runDefault, j, errJson := serializeV2Block(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &bellatrixBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data.Message)
		beaconBlock := resp.Data.Message
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		assert.NotNil(t, beaconBlock.Body)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})

	t.Run("incorrect response type", func(t *testing.T) {
		response := &types.Empty{}
		runDefault, j, errJson := serializeV2Block(response)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "container is not of the correct type"))
	})

	t.Run("unsupported block version", func(t *testing.T) {
		response := &BlockV2ResponseJson{
			Version: "unsupported",
		}
		runDefault, j, errJson := serializeV2Block(response)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "unsupported block version"))
	})
}

func TestSerializeBlindedBlock(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		response := &BlindedBlockResponseJson{
			Version: ethpbv2.Version_PHASE0.String(),
			Data: &SignedBlindedBeaconBlockContainerJson{
				Phase0Block: &BeaconBlockJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyJson{},
				},
				Signature: "sig",
			},
			ExecutionOptimistic: true,
		}
		runDefault, j, errJson := serializeBlindedBlock(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &phase0BlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data.Message)
		beaconBlock := resp.Data.Message
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		assert.NotNil(t, beaconBlock.Body)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})

	t.Run("Altair", func(t *testing.T) {
		response := &BlindedBlockResponseJson{
			Version: ethpbv2.Version_ALTAIR.String(),
			Data: &SignedBlindedBeaconBlockContainerJson{
				AltairBlock: &BeaconBlockAltairJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyAltairJson{},
				},
				Signature: "sig",
			},
			ExecutionOptimistic: true,
		}
		runDefault, j, errJson := serializeBlindedBlock(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &altairBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data.Message)
		beaconBlock := resp.Data.Message
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		assert.NotNil(t, beaconBlock.Body)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		response := &BlindedBlockResponseJson{
			Version: ethpbv2.Version_BELLATRIX.String(),
			Data: &SignedBlindedBeaconBlockContainerJson{
				BellatrixBlock: &BlindedBeaconBlockBellatrixJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BlindedBeaconBlockBodyBellatrixJson{},
				},
				Signature: "sig",
			},
			ExecutionOptimistic: true,
		}
		runDefault, j, errJson := serializeBlindedBlock(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &bellatrixBlindedBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data.Message)
		beaconBlock := resp.Data.Message
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		assert.NotNil(t, beaconBlock.Body)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})

	t.Run("Capella", func(t *testing.T) {
		response := &BlindedBlockResponseJson{
			Version: ethpbv2.Version_CAPELLA.String(),
			Data: &SignedBlindedBeaconBlockContainerJson{
				CapellaBlock: &BlindedBeaconBlockCapellaJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body: &BlindedBeaconBlockBodyCapellaJson{
						ExecutionPayloadHeader: &ExecutionPayloadHeaderCapellaJson{
							ParentHash:       "parent_hash",
							FeeRecipient:     "fee_recipient",
							StateRoot:        "state_root",
							ReceiptsRoot:     "receipts_root",
							LogsBloom:        "logs_bloom",
							PrevRandao:       "prev_randao",
							BlockNumber:      "block_number",
							GasLimit:         "gas_limit",
							GasUsed:          "gas_used",
							TimeStamp:        "time_stamp",
							ExtraData:        "extra_data",
							BaseFeePerGas:    "base_fee_per_gas",
							BlockHash:        "block_hash",
							TransactionsRoot: "transactions_root",
							WithdrawalsRoot:  "withdrawals_root",
						},
					},
				},
				Signature: "sig",
			},
			ExecutionOptimistic: true,
		}
		runDefault, j, errJson := serializeBlindedBlock(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &capellaBlindedBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data.Message)
		beaconBlock := resp.Data.Message
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		assert.NotNil(t, beaconBlock.Body)
		payloadHeader := beaconBlock.Body.ExecutionPayloadHeader
		assert.NotNil(t, payloadHeader)
		assert.Equal(t, "parent_hash", payloadHeader.ParentHash)
		assert.Equal(t, "fee_recipient", payloadHeader.FeeRecipient)
		assert.Equal(t, "state_root", payloadHeader.StateRoot)
		assert.Equal(t, "receipts_root", payloadHeader.ReceiptsRoot)
		assert.Equal(t, "logs_bloom", payloadHeader.LogsBloom)
		assert.Equal(t, "prev_randao", payloadHeader.PrevRandao)
		assert.Equal(t, "block_number", payloadHeader.BlockNumber)
		assert.Equal(t, "gas_limit", payloadHeader.GasLimit)
		assert.Equal(t, "gas_used", payloadHeader.GasUsed)
		assert.Equal(t, "time_stamp", payloadHeader.TimeStamp)
		assert.Equal(t, "extra_data", payloadHeader.ExtraData)
		assert.Equal(t, "base_fee_per_gas", payloadHeader.BaseFeePerGas)
		assert.Equal(t, "block_hash", payloadHeader.BlockHash)
		assert.Equal(t, "transactions_root", payloadHeader.TransactionsRoot)
		assert.Equal(t, "withdrawals_root", payloadHeader.WithdrawalsRoot)
		assert.Equal(t, true, resp.ExecutionOptimistic)

	})

	t.Run("incorrect response type", func(t *testing.T) {
		response := &types.Empty{}
		runDefault, j, errJson := serializeBlindedBlock(response)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "container is not of the correct type"))
	})

	t.Run("unsupported block version", func(t *testing.T) {
		response := &BlindedBlockResponseJson{
			Version: "unsupported",
		}
		runDefault, j, errJson := serializeBlindedBlock(response)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "unsupported block version"))
	})
}

func TestSerializeV2State(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		response := &BeaconStateV2ResponseJson{
			Version: ethpbv2.Version_PHASE0.String(),
			Data: &BeaconStateContainerV2Json{
				Phase0State: &BeaconStateJson{},
				AltairState: nil,
			},
		}
		runDefault, j, errJson := serializeV2State(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		require.NoError(t, json.Unmarshal(j, &phase0StateResponseJson{}))
	})

	t.Run("Altair", func(t *testing.T) {
		response := &BeaconStateV2ResponseJson{
			Version: ethpbv2.Version_ALTAIR.String(),
			Data: &BeaconStateContainerV2Json{
				Phase0State: nil,
				AltairState: &BeaconStateAltairJson{},
			},
		}
		runDefault, j, errJson := serializeV2State(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		require.NoError(t, json.Unmarshal(j, &altairStateResponseJson{}))
	})

	t.Run("Bellatrix", func(t *testing.T) {
		response := &BeaconStateV2ResponseJson{
			Version: ethpbv2.Version_BELLATRIX.String(),
			Data: &BeaconStateContainerV2Json{
				Phase0State:    nil,
				BellatrixState: &BeaconStateBellatrixJson{},
			},
		}
		runDefault, j, errJson := serializeV2State(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		require.NoError(t, json.Unmarshal(j, &bellatrixStateResponseJson{}))
	})

	t.Run("Capella", func(t *testing.T) {
		response := &BeaconStateV2ResponseJson{
			Version: ethpbv2.Version_CAPELLA.String(),
			Data: &BeaconStateContainerV2Json{
				Phase0State:  nil,
				CapellaState: &BeaconStateCapellaJson{},
			},
		}
		runDefault, j, errJson := serializeV2State(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		require.NoError(t, json.Unmarshal(j, &capellaStateResponseJson{}))
	})

	t.Run("incorrect response type", func(t *testing.T) {
		runDefault, j, errJson := serializeV2State(&types.Empty{})
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "container is not of the correct type"))
	})

	t.Run("unsupported state version", func(t *testing.T) {
		response := &BeaconStateV2ResponseJson{
			Version: "unsupported",
		}
		runDefault, j, errJson := serializeV2State(response)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "unsupported state version"))
	})
}

func TestSerializeProducedV2Block(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		response := &ProduceBlockResponseV2Json{
			Version: ethpbv2.Version_PHASE0.String(),
			Data: &BeaconBlockContainerV2Json{
				Phase0Block: &BeaconBlockJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyJson{},
				},
			},
		}
		runDefault, j, errJson := serializeProducedV2Block(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &phase0ProduceBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data)
		beaconBlock := resp.Data
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		require.NotNil(t, beaconBlock.Body)
	})

	t.Run("Altair", func(t *testing.T) {
		response := &ProduceBlockResponseV2Json{
			Version: ethpbv2.Version_ALTAIR.String(),
			Data: &BeaconBlockContainerV2Json{
				AltairBlock: &BeaconBlockAltairJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyAltairJson{},
				},
			},
		}
		runDefault, j, errJson := serializeProducedV2Block(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &altairProduceBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data)
		beaconBlock := resp.Data
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		require.NotNil(t, beaconBlock.Body)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		response := &ProduceBlockResponseV2Json{
			Version: ethpbv2.Version_BELLATRIX.String(),
			Data: &BeaconBlockContainerV2Json{
				BellatrixBlock: &BeaconBlockBellatrixJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyBellatrixJson{},
				},
			},
		}
		runDefault, j, errJson := serializeProducedV2Block(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &capellaProduceBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data)
		beaconBlock := resp.Data
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		require.NotNil(t, beaconBlock.Body)
	})
	t.Run("Capella", func(t *testing.T) {
		response := &ProduceBlockResponseV2Json{
			Version: ethpbv2.Version_CAPELLA.String(),
			Data: &BeaconBlockContainerV2Json{
				CapellaBlock: &BeaconBlockCapellaJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyCapellaJson{},
				},
			},
		}
		runDefault, j, errJson := serializeProducedV2Block(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &capellaProduceBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data)
		beaconBlock := resp.Data
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		require.NotNil(t, beaconBlock.Body)
	})
	t.Run("incorrect response type", func(t *testing.T) {
		response := &types.Empty{}
		runDefault, j, errJson := serializeProducedV2Block(response)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "container is not of the correct type"))
	})

	t.Run("unsupported block version", func(t *testing.T) {
		response := &ProduceBlockResponseV2Json{
			Version: "unsupported",
		}
		runDefault, j, errJson := serializeProducedV2Block(response)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "unsupported block version"))
	})
}

func TestSerializeProduceBlindedBlock(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		response := &ProduceBlindedBlockResponseJson{
			Version: ethpbv2.Version_PHASE0.String(),
			Data: &BlindedBeaconBlockContainerJson{
				Phase0Block: &BeaconBlockJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyJson{},
				},
				AltairBlock: nil,
			},
		}
		runDefault, j, errJson := serializeProducedBlindedBlock(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &phase0ProduceBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data)
		beaconBlock := resp.Data
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		require.NotNil(t, beaconBlock.Body)
	})

	t.Run("Altair", func(t *testing.T) {
		response := &ProduceBlindedBlockResponseJson{
			Version: ethpbv2.Version_ALTAIR.String(),
			Data: &BlindedBeaconBlockContainerJson{
				AltairBlock: &BeaconBlockAltairJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BeaconBlockBodyAltairJson{},
				},
			},
		}
		runDefault, j, errJson := serializeProducedBlindedBlock(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &altairProduceBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data)
		beaconBlock := resp.Data
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		require.NotNil(t, beaconBlock.Body)
	})

	t.Run("Bellatrix", func(t *testing.T) {
		response := &ProduceBlindedBlockResponseJson{
			Version: ethpbv2.Version_BELLATRIX.String(),
			Data: &BlindedBeaconBlockContainerJson{
				BellatrixBlock: &BlindedBeaconBlockBellatrixJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BlindedBeaconBlockBodyBellatrixJson{},
				},
			},
		}
		runDefault, j, errJson := serializeProducedBlindedBlock(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &bellatrixProduceBlindedBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data)
		beaconBlock := resp.Data
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		require.NotNil(t, beaconBlock.Body)
	})

	t.Run("Capella", func(t *testing.T) {
		response := &ProduceBlindedBlockResponseJson{
			Version: ethpbv2.Version_CAPELLA.String(),
			Data: &BlindedBeaconBlockContainerJson{
				CapellaBlock: &BlindedBeaconBlockCapellaJson{
					Slot:          "1",
					ProposerIndex: "1",
					ParentRoot:    "root",
					StateRoot:     "root",
					Body:          &BlindedBeaconBlockBodyCapellaJson{},
				},
			},
		}
		runDefault, j, errJson := serializeProducedBlindedBlock(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &capellaProduceBlindedBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		beaconBlock := resp.Data
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		require.NotNil(t, beaconBlock.Body)
	})

	t.Run("incorrect response type", func(t *testing.T) {
		response := &types.Empty{}
		runDefault, j, errJson := serializeProducedV2Block(response)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "container is not of the correct type"))
	})

	t.Run("unsupported block version", func(t *testing.T) {
		response := &ProduceBlockResponseV2Json{
			Version: "unsupported",
		}
		runDefault, j, errJson := serializeProducedV2Block(response)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.Equal(t, 0, len(j))
		require.NotNil(t, errJson)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "unsupported block version"))
	})
}

func TestPrepareForkChoiceResponse(t *testing.T) {
	dump := &ForkChoiceDumpJson{
		JustifiedCheckpoint: &CheckpointJson{
			Epoch: "justified",
			Root:  "justified",
		},
		FinalizedCheckpoint: &CheckpointJson{
			Epoch: "finalized",
			Root:  "finalized",
		},
		BestJustifiedCheckpoint: &CheckpointJson{
			Epoch: "best_justified",
			Root:  "best_justified",
		},
		UnrealizedJustifiedCheckpoint: &CheckpointJson{
			Epoch: "unrealized_justified",
			Root:  "unrealized_justified",
		},
		UnrealizedFinalizedCheckpoint: &CheckpointJson{
			Epoch: "unrealized_finalized",
			Root:  "unrealized_finalized",
		},
		ProposerBoostRoot:         "proposer_boost_root",
		PreviousProposerBoostRoot: "previous_proposer_boost_root",
		HeadRoot:                  "head_root",
		ForkChoiceNodes: []*ForkChoiceNodeJson{
			{
				Slot:                     "node1_slot",
				BlockRoot:                "node1_block_root",
				ParentRoot:               "node1_parent_root",
				JustifiedEpoch:           "node1_justified_epoch",
				FinalizedEpoch:           "node1_finalized_epoch",
				UnrealizedJustifiedEpoch: "node1_unrealized_justified_epoch",
				UnrealizedFinalizedEpoch: "node1_unrealized_finalized_epoch",
				Balance:                  "node1_balance",
				Weight:                   "node1_weight",
				ExecutionOptimistic:      false,
				ExecutionBlockHash:       "node1_execution_block_hash",
				TimeStamp:                "node1_time_stamp",
				Validity:                 "node1_validity",
			},
			{
				Slot:                     "node2_slot",
				BlockRoot:                "node2_block_root",
				ParentRoot:               "node2_parent_root",
				JustifiedEpoch:           "node2_justified_epoch",
				FinalizedEpoch:           "node2_finalized_epoch",
				UnrealizedJustifiedEpoch: "node2_unrealized_justified_epoch",
				UnrealizedFinalizedEpoch: "node2_unrealized_finalized_epoch",
				Balance:                  "node2_balance",
				Weight:                   "node2_weight",
				ExecutionOptimistic:      true,
				ExecutionBlockHash:       "node2_execution_block_hash",
				TimeStamp:                "node2_time_stamp",
				Validity:                 "node2_validity",
			},
		},
	}
	runDefault, j, errorJson := prepareForkChoiceResponse(dump)
	assert.Equal(t, nil, errorJson)
	assert.Equal(t, apimiddleware.RunDefault(false), runDefault)
	result := &ForkChoiceResponseJson{}
	require.NoError(t, json.Unmarshal(j, result))
	require.NotNil(t, result)
	assert.Equal(t, "justified", result.JustifiedCheckpoint.Epoch)
	assert.Equal(t, "justified", result.JustifiedCheckpoint.Root)
	assert.Equal(t, "finalized", result.FinalizedCheckpoint.Epoch)
	assert.Equal(t, "finalized", result.FinalizedCheckpoint.Root)
	assert.Equal(t, "best_justified", result.ExtraData.BestJustifiedCheckpoint.Epoch)
	assert.Equal(t, "best_justified", result.ExtraData.BestJustifiedCheckpoint.Root)
	assert.Equal(t, "unrealized_justified", result.ExtraData.UnrealizedJustifiedCheckpoint.Epoch)
	assert.Equal(t, "unrealized_justified", result.ExtraData.UnrealizedJustifiedCheckpoint.Root)
	assert.Equal(t, "unrealized_finalized", result.ExtraData.UnrealizedFinalizedCheckpoint.Epoch)
	assert.Equal(t, "unrealized_finalized", result.ExtraData.UnrealizedFinalizedCheckpoint.Root)
	assert.Equal(t, "proposer_boost_root", result.ExtraData.ProposerBoostRoot)
	assert.Equal(t, "previous_proposer_boost_root", result.ExtraData.PreviousProposerBoostRoot)
	assert.Equal(t, "head_root", result.ExtraData.HeadRoot)
	require.Equal(t, 2, len(result.ForkChoiceNodes))
	node1 := result.ForkChoiceNodes[0]
	require.NotNil(t, node1)
	assert.Equal(t, "node1_slot", node1.Slot)
	assert.Equal(t, "node1_block_root", node1.BlockRoot)
	assert.Equal(t, "node1_parent_root", node1.ParentRoot)
	assert.Equal(t, "node1_justified_epoch", node1.JustifiedEpoch)
	assert.Equal(t, "node1_finalized_epoch", node1.FinalizedEpoch)
	assert.Equal(t, "node1_unrealized_justified_epoch", node1.ExtraData.UnrealizedJustifiedEpoch)
	assert.Equal(t, "node1_unrealized_finalized_epoch", node1.ExtraData.UnrealizedFinalizedEpoch)
	assert.Equal(t, "node1_balance", node1.ExtraData.Balance)
	assert.Equal(t, "node1_weight", node1.Weight)
	assert.Equal(t, false, node1.ExtraData.ExecutionOptimistic)
	assert.Equal(t, "node1_execution_block_hash", node1.ExecutionBlockHash)
	assert.Equal(t, "node1_time_stamp", node1.ExtraData.TimeStamp)
	assert.Equal(t, "node1_validity", node1.Validity)
	node2 := result.ForkChoiceNodes[1]
	require.NotNil(t, node2)
	assert.Equal(t, "node2_slot", node2.Slot)
	assert.Equal(t, "node2_block_root", node2.BlockRoot)
	assert.Equal(t, "node2_parent_root", node2.ParentRoot)
	assert.Equal(t, "node2_justified_epoch", node2.JustifiedEpoch)
	assert.Equal(t, "node2_finalized_epoch", node2.FinalizedEpoch)
	assert.Equal(t, "node2_unrealized_justified_epoch", node2.ExtraData.UnrealizedJustifiedEpoch)
	assert.Equal(t, "node2_unrealized_finalized_epoch", node2.ExtraData.UnrealizedFinalizedEpoch)
	assert.Equal(t, "node2_balance", node2.ExtraData.Balance)
	assert.Equal(t, "node2_weight", node2.Weight)
	assert.Equal(t, true, node2.ExtraData.ExecutionOptimistic)
	assert.Equal(t, "node2_execution_block_hash", node2.ExecutionBlockHash)
	assert.Equal(t, "node2_time_stamp", node2.ExtraData.TimeStamp)
	assert.Equal(t, "node2_validity", node2.Validity)
}
