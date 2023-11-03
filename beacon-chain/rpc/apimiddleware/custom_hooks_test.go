package apimiddleware

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func TestSetInitialPublishBlockPostRequest(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.BellatrixForkEpoch = params.BeaconConfig().AltairForkEpoch + 1
	cfg.CapellaForkEpoch = params.BeaconConfig().BellatrixForkEpoch + 1
	cfg.DenebForkEpoch = params.BeaconConfig().CapellaForkEpoch + 1
	params.OverrideBeaconConfig(cfg)

	endpoint := &apimiddleware.Endpoint{}
	s := struct {
		Message struct {
			Slot string
		} `json:"message"`
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
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
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
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockAltairJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
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
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockBellatrixJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
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
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockCapellaJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
	t.Run("Deneb", func(t *testing.T) {
		params.SetupTestConfigCleanup(t)
		cfg := params.BeaconConfig()
		cfg.DenebForkEpoch = cfg.CapellaForkEpoch.Add(2)
		params.OverrideBeaconConfig(cfg)
		slot, err := slots.EpochStart(params.BeaconConfig().DenebForkEpoch)
		require.NoError(t, err)
		denebS := struct {
			SignedBlock struct {
				Message struct {
					Slot string
				} `json:"message"`
			} `json:"signed_block"`
		}{}
		denebS.SignedBlock = struct {
			Message struct {
				Slot string
			} `json:"message"`
		}{
			Message: struct {
				Slot string
			}{Slot: strconv.FormatUint(uint64(slot), 10)},
		}
		j, err := json.Marshal(denebS)
		require.NoError(t, err)
		var body bytes.Buffer
		_, err = body.Write(j)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		runDefault, errJson := setInitialPublishBlockPostRequest(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockContentsDenebJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
}

func TestPreparePublishedBlock(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SignedBeaconBlockJson{
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
			PostRequest: &SignedBeaconBlockAltairJson{
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
			PostRequest: &SignedBeaconBlockBellatrixJson{
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
	cfg.CapellaForkEpoch = params.BeaconConfig().BellatrixForkEpoch + 1
	cfg.DenebForkEpoch = params.BeaconConfig().CapellaForkEpoch + 1
	params.OverrideBeaconConfig(cfg)

	endpoint := &apimiddleware.Endpoint{}
	s := struct {
		Message struct {
			Slot string
		} `json:"message"`
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
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
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
		assert.Equal(t, reflect.TypeOf(SignedBeaconBlockAltairJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
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
		assert.Equal(t, reflect.TypeOf(SignedBlindedBeaconBlockBellatrixJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
	t.Run("Capella", func(t *testing.T) {
		slot, err := slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
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
		assert.Equal(t, reflect.TypeOf(SignedBlindedBeaconBlockCapellaJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
	t.Run("Deneb", func(t *testing.T) {
		slot, err := slots.EpochStart(params.BeaconConfig().DenebForkEpoch)
		require.NoError(t, err)
		denebS := struct {
			SignedBlindedBlock struct {
				Message struct {
					Slot string
				} `json:"message"`
			} `json:"signed_blinded_block"`
		}{}
		denebS.SignedBlindedBlock = struct {
			Message struct {
				Slot string
			} `json:"message"`
		}{
			Message: struct {
				Slot string
			}{Slot: strconv.FormatUint(uint64(slot), 10)},
		}
		j, err := json.Marshal(denebS)
		require.NoError(t, err)
		var body bytes.Buffer
		_, err = body.Write(j)
		require.NoError(t, err)
		request := httptest.NewRequest("POST", "http://foo.example", &body)
		runDefault, errJson := setInitialPublishBlindedBlockPostRequest(endpoint, nil, request)
		require.Equal(t, true, errJson == nil)
		assert.Equal(t, apimiddleware.RunDefault(true), runDefault)
		assert.Equal(t, reflect.TypeOf(SignedBlindedBeaconBlockContentsDenebJson{}).Name(), reflect.Indirect(reflect.ValueOf(endpoint.PostRequest)).Type().Name())
	})
}

func TestPreparePublishedBlindedBlock(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SignedBeaconBlockJson{
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
			PostRequest: &SignedBeaconBlockAltairJson{
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
			PostRequest: &SignedBlindedBeaconBlockBellatrixJson{
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
	t.Run("Capella", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SignedBlindedBeaconBlockCapellaJson{
				Message: &BlindedBeaconBlockCapellaJson{
					Body: &BlindedBeaconBlockBodyCapellaJson{},
				},
			},
		}
		errJson := preparePublishedBlindedBlock(endpoint, nil, nil)
		require.Equal(t, true, errJson == nil)
		_, ok := endpoint.PostRequest.(*capellaPublishBlindedBlockRequestJson)
		assert.Equal(t, true, ok)
	})

	t.Run("Deneb", func(t *testing.T) {
		endpoint := &apimiddleware.Endpoint{
			PostRequest: &SignedBlindedBeaconBlockContentsDenebJson{
				SignedBlindedBlock: &SignedBlindedBeaconBlockDenebJson{
					Message: &BlindedBeaconBlockDenebJson{},
				},
				SignedBlindedBlobSidecars: []*SignedBlindedBlobSidecarJson{},
			},
		}
		errJson := preparePublishedBlindedBlock(endpoint, nil, nil)
		require.Equal(t, true, errJson == nil)
		_, ok := endpoint.PostRequest.(*denebPublishBlindedBlockRequestJson)
		assert.Equal(t, true, ok)
	})
	t.Run("unsupported block type", func(t *testing.T) {
		errJson := preparePublishedBlock(&apimiddleware.Endpoint{}, nil, nil)
		assert.Equal(t, true, strings.Contains(errJson.Msg(), "unsupported block type"))
	})
}
