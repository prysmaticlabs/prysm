package apimiddleware

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
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
	t.Run("Deneb", func(t *testing.T) {
		response := &ProduceBlockResponseV2Json{
			Version: ethpbv2.Version_DENEB.String(),
			Data: &BeaconBlockContainerV2Json{
				DenebContents: &BeaconBlockContentsDenebJson{
					Block: &BeaconBlockDenebJson{
						Slot:          "1",
						ProposerIndex: "1",
						ParentRoot:    "root",
						StateRoot:     "root",
						Body:          &BeaconBlockBodyDenebJson{},
					},
					BlobSidecars: []*BlobSidecarJson{{
						BlockRoot:       "root",
						Index:           "1",
						Slot:            "1",
						BlockParentRoot: "root",
						ProposerIndex:   "1",
						Blob:            "blob",
						KzgCommitment:   "kzgcommitment",
						KzgProof:        "kzgproof",
					}},
				},
			},
		}
		runDefault, j, errJson := serializeProducedV2Block(response)
		require.Equal(t, nil, errJson)
		require.Equal(t, apimiddleware.RunDefault(false), runDefault)
		require.NotNil(t, j)
		resp := &denebProduceBlockResponseJson{}
		require.NoError(t, json.Unmarshal(j, resp))
		require.NotNil(t, resp.Data)
		require.NotNil(t, resp.Data)
		beaconBlock := resp.Data.Block
		assert.Equal(t, "1", beaconBlock.Slot)
		assert.Equal(t, "1", beaconBlock.ProposerIndex)
		assert.Equal(t, "root", beaconBlock.ParentRoot)
		assert.Equal(t, "root", beaconBlock.StateRoot)
		assert.NotNil(t, beaconBlock.Body)
		require.Equal(t, 1, len(resp.Data.BlobSidecars))
		sidecar := resp.Data.BlobSidecars[0]
		assert.Equal(t, "root", sidecar.BlockRoot)
		assert.Equal(t, "1", sidecar.Index)
		assert.Equal(t, "1", sidecar.Slot)
		assert.Equal(t, "root", sidecar.BlockParentRoot)
		assert.Equal(t, "1", sidecar.ProposerIndex)
		assert.Equal(t, "blob", sidecar.Blob)
		assert.Equal(t, "kzgcommitment", sidecar.KzgCommitment)
		assert.Equal(t, "kzgproof", sidecar.KzgProof)
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
