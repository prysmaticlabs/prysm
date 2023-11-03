package apimiddleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// https://ethereum.github.io/beacon-APIs/?urls.primaryName=dev#/Beacon/submitPoolBLSToExecutionChange
// expects posting a top-level array. We make it more proto-friendly by wrapping it in a struct.

type v1alpha1SignedPhase0Block struct {
	Block     *BeaconBlockJson `json:"block"` // tech debt on phase 0 called this block instead of "message"
	Signature string           `json:"signature" hex:"true"`
}

type phase0PublishBlockRequestJson struct {
	Message *v1alpha1SignedPhase0Block `json:"phase0_block"`
}

type altairPublishBlockRequestJson struct {
	AltairBlock *SignedBeaconBlockAltairJson `json:"altair_block"`
}

type bellatrixPublishBlockRequestJson struct {
	BellatrixBlock *SignedBeaconBlockBellatrixJson `json:"bellatrix_block"`
}

type bellatrixPublishBlindedBlockRequestJson struct {
	BellatrixBlock *SignedBlindedBeaconBlockBellatrixJson `json:"bellatrix_block"`
}

type capellaPublishBlockRequestJson struct {
	CapellaBlock *SignedBeaconBlockCapellaJson `json:"capella_block"`
}

type capellaPublishBlindedBlockRequestJson struct {
	CapellaBlock *SignedBlindedBeaconBlockCapellaJson `json:"capella_block"`
}

type denebPublishBlockRequestJson struct {
	DenebContents *SignedBeaconBlockContentsDenebJson `json:"deneb_contents"`
}

type denebPublishBlindedBlockRequestJson struct {
	DenebContents *SignedBlindedBeaconBlockContentsDenebJson `json:"deneb_contents"`
}

// setInitialPublishBlockPostRequest is triggered before we deserialize the request JSON into a struct.
// We don't know which version of the block got posted, but we can determine it from the slot.
// We know that blocks of all versions have a Message field with a Slot field,
// so we deserialize the request into a struct s, which has the right fields, to obtain the slot.
// Once we know the slot, we can determine what the PostRequest field of the endpoint should be, and we set it appropriately.
func setInitialPublishBlockPostRequest(endpoint *apimiddleware.Endpoint,
	_ http.ResponseWriter,
	req *http.Request,
) (apimiddleware.RunDefault, apimiddleware.ErrorJson) {
	s := struct {
		Slot string
	}{}

	buf, err := io.ReadAll(req.Body)
	if err != nil {
		return false, apimiddleware.InternalServerErrorWithMessage(err, "could not read body")
	}

	typeParseMap := make(map[string]json.RawMessage)
	if err := json.Unmarshal(buf, &typeParseMap); err != nil {
		return false, apimiddleware.InternalServerErrorWithMessage(err, "could not parse object")
	}
	if val, ok := typeParseMap["message"]; ok {
		if err := json.Unmarshal(val, &s); err != nil {
			return false, apimiddleware.InternalServerErrorWithMessage(err, "could not unmarshal field 'message' ")
		}
	} else if val, ok := typeParseMap["signed_block"]; ok {
		temp := struct {
			Message struct {
				Slot string
			}
		}{}
		if err := json.Unmarshal(val, &temp); err != nil {
			return false, apimiddleware.InternalServerErrorWithMessage(err, "could not unmarshal field 'signed_block' ")
		}
		s.Slot = temp.Message.Slot
	} else {
		return false, &apimiddleware.DefaultErrorJson{Message: "could not parse slot from request", Code: http.StatusInternalServerError}
	}
	slot, err := strconv.ParseUint(s.Slot, 10, 64)
	if err != nil {
		return false, apimiddleware.InternalServerErrorWithMessage(err, "slot is not an unsigned integer")
	}
	currentEpoch := slots.ToEpoch(primitives.Slot(slot))
	if currentEpoch < params.BeaconConfig().AltairForkEpoch {
		endpoint.PostRequest = &SignedBeaconBlockJson{}
	} else if currentEpoch < params.BeaconConfig().BellatrixForkEpoch {
		endpoint.PostRequest = &SignedBeaconBlockAltairJson{}
	} else if currentEpoch < params.BeaconConfig().CapellaForkEpoch {
		endpoint.PostRequest = &SignedBeaconBlockBellatrixJson{}
	} else if currentEpoch < params.BeaconConfig().DenebForkEpoch {
		endpoint.PostRequest = &SignedBeaconBlockCapellaJson{}
	} else {
		endpoint.PostRequest = &SignedBeaconBlockContentsDenebJson{}
	}
	req.Body = io.NopCloser(bytes.NewBuffer(buf))
	return true, nil
}

// In preparePublishedBlock we transform the PostRequest.
// gRPC expects an XXX_block field in the JSON object, but we have a message field at this point.
// We do a simple conversion depending on the type of endpoint.PostRequest
// (which was filled out previously in setInitialPublishBlockPostRequest).
func preparePublishedBlock(endpoint *apimiddleware.Endpoint, _ http.ResponseWriter, _ *http.Request) apimiddleware.ErrorJson {
	if block, ok := endpoint.PostRequest.(*SignedBeaconBlockJson); ok {
		// Prepare post request that can be properly decoded on gRPC side.
		endpoint.PostRequest = &phase0PublishBlockRequestJson{
			Message: &v1alpha1SignedPhase0Block{
				Block:     block.Message,
				Signature: block.Signature,
			},
		}
		return nil
	}
	if block, ok := endpoint.PostRequest.(*SignedBeaconBlockAltairJson); ok {
		// Prepare post request that can be properly decoded on gRPC side.
		endpoint.PostRequest = &altairPublishBlockRequestJson{
			AltairBlock: block,
		}
		return nil
	}
	if block, ok := endpoint.PostRequest.(*SignedBeaconBlockBellatrixJson); ok {
		// Prepare post request that can be properly decoded on gRPC side.
		endpoint.PostRequest = &bellatrixPublishBlockRequestJson{
			BellatrixBlock: block,
		}
		return nil
	}
	if block, ok := endpoint.PostRequest.(*SignedBeaconBlockCapellaJson); ok {
		// Prepare post request that can be properly decoded on gRPC side.
		endpoint.PostRequest = &capellaPublishBlockRequestJson{
			CapellaBlock: block,
		}
		return nil
	}
	if block, ok := endpoint.PostRequest.(*SignedBeaconBlockContentsDenebJson); ok {
		// Prepare post request that can be properly decoded on gRPC side.
		endpoint.PostRequest = &denebPublishBlockRequestJson{
			DenebContents: block,
		}
		return nil
	}
	return apimiddleware.InternalServerError(errors.New("unsupported block type"))
}

// setInitialPublishBlindedBlockPostRequest is triggered before we deserialize the request JSON into a struct.
// We don't know which version of the block got posted, but we can determine it from the slot.
// We know that blocks of all versions have a Message field with a Slot field,
// so we deserialize the request into a struct s, which has the right fields, to obtain the slot.
// Once we know the slot, we can determine what the PostRequest field of the endpoint should be, and we set it appropriately.
func setInitialPublishBlindedBlockPostRequest(endpoint *apimiddleware.Endpoint,
	_ http.ResponseWriter,
	req *http.Request,
) (apimiddleware.RunDefault, apimiddleware.ErrorJson) {
	s := struct {
		Slot string
	}{}

	buf, err := io.ReadAll(req.Body)
	if err != nil {
		return false, apimiddleware.InternalServerErrorWithMessage(err, "could not read body")
	}

	typeParseMap := make(map[string]json.RawMessage)
	if err = json.Unmarshal(buf, &typeParseMap); err != nil {
		return false, apimiddleware.InternalServerErrorWithMessage(err, "could not parse object")
	}
	if val, ok := typeParseMap["message"]; ok {
		if err = json.Unmarshal(val, &s); err != nil {
			return false, apimiddleware.InternalServerErrorWithMessage(err, "could not unmarshal field 'message' ")
		}
	} else if val, ok = typeParseMap["signed_blinded_block"]; ok {
		temp := struct {
			Message struct {
				Slot string
			}
		}{}
		if err = json.Unmarshal(val, &temp); err != nil {
			return false, apimiddleware.InternalServerErrorWithMessage(err, "could not unmarshal field 'signed_block' ")
		}
		s.Slot = temp.Message.Slot
	} else {
		return false, &apimiddleware.DefaultErrorJson{Message: "could not parse slot from request", Code: http.StatusInternalServerError}
	}
	slot, err := strconv.ParseUint(s.Slot, 10, 64)
	if err != nil {
		return false, apimiddleware.InternalServerErrorWithMessage(err, "slot is not an unsigned integer")
	}
	currentEpoch := slots.ToEpoch(primitives.Slot(slot))
	if currentEpoch < params.BeaconConfig().AltairForkEpoch {
		endpoint.PostRequest = &SignedBeaconBlockJson{}
	} else if currentEpoch < params.BeaconConfig().BellatrixForkEpoch {
		endpoint.PostRequest = &SignedBeaconBlockAltairJson{}
	} else if currentEpoch < params.BeaconConfig().CapellaForkEpoch {
		endpoint.PostRequest = &SignedBlindedBeaconBlockBellatrixJson{}
	} else if currentEpoch < params.BeaconConfig().DenebForkEpoch {
		endpoint.PostRequest = &SignedBlindedBeaconBlockCapellaJson{}
	} else {
		endpoint.PostRequest = &SignedBlindedBeaconBlockContentsDenebJson{}
	}
	req.Body = io.NopCloser(bytes.NewBuffer(buf))
	return true, nil
}

// In preparePublishedBlindedBlock we transform the PostRequest.
// gRPC expects either an XXX_block field in the JSON object, but we have a message field at this point.
// We do a simple conversion depending on the type of endpoint.PostRequest
// (which was filled out previously in setInitialPublishBlockPostRequest).
func preparePublishedBlindedBlock(endpoint *apimiddleware.Endpoint, _ http.ResponseWriter, _ *http.Request) apimiddleware.ErrorJson {
	if block, ok := endpoint.PostRequest.(*SignedBeaconBlockJson); ok {
		endpoint.PostRequest = &phase0PublishBlockRequestJson{
			Message: &v1alpha1SignedPhase0Block{
				Block:     block.Message,
				Signature: block.Signature,
			},
		}
		return nil
	}
	if block, ok := endpoint.PostRequest.(*SignedBeaconBlockAltairJson); ok {
		// Prepare post request that can be properly decoded on gRPC side.
		actualPostReq := &altairPublishBlockRequestJson{
			AltairBlock: block,
		}
		endpoint.PostRequest = actualPostReq
		return nil
	}
	if block, ok := endpoint.PostRequest.(*SignedBlindedBeaconBlockBellatrixJson); ok {
		// Prepare post request that can be properly decoded on gRPC side.
		actualPostReq := &bellatrixPublishBlindedBlockRequestJson{
			BellatrixBlock: &SignedBlindedBeaconBlockBellatrixJson{
				Message:   block.Message,
				Signature: block.Signature,
			},
		}
		endpoint.PostRequest = actualPostReq
		return nil
	}
	if block, ok := endpoint.PostRequest.(*SignedBlindedBeaconBlockCapellaJson); ok {
		// Prepare post request that can be properly decoded on gRPC side.
		actualPostReq := &capellaPublishBlindedBlockRequestJson{
			CapellaBlock: &SignedBlindedBeaconBlockCapellaJson{
				Message:   block.Message,
				Signature: block.Signature,
			},
		}
		endpoint.PostRequest = actualPostReq
		return nil
	}
	if blockContents, ok := endpoint.PostRequest.(*SignedBlindedBeaconBlockContentsDenebJson); ok {
		// Prepare post request that can be properly decoded on gRPC side.
		actualPostReq := &denebPublishBlindedBlockRequestJson{
			DenebContents: &SignedBlindedBeaconBlockContentsDenebJson{
				SignedBlindedBlock:        blockContents.SignedBlindedBlock,
				SignedBlindedBlobSidecars: blockContents.SignedBlindedBlobSidecars,
			},
		}
		endpoint.PostRequest = actualPostReq
		return nil
	}
	return apimiddleware.InternalServerError(errors.New("unsupported block type"))
}

type phase0ProduceBlockResponseJson struct {
	Version string           `json:"version" enum:"true"`
	Data    *BeaconBlockJson `json:"data"`
}

type altairProduceBlockResponseJson struct {
	Version string                 `json:"version" enum:"true"`
	Data    *BeaconBlockAltairJson `json:"data"`
}

type bellatrixProduceBlockResponseJson struct {
	Version string                    `json:"version" enum:"true"`
	Data    *BeaconBlockBellatrixJson `json:"data"`
}

type capellaProduceBlockResponseJson struct {
	Version string                  `json:"version" enum:"true"`
	Data    *BeaconBlockCapellaJson `json:"data"`
}

type denebProduceBlockResponseJson struct {
	Version string                        `json:"version" enum:"true"`
	Data    *BeaconBlockContentsDenebJson `json:"data"`
}

type bellatrixProduceBlindedBlockResponseJson struct {
	Version string                           `json:"version" enum:"true"`
	Data    *BlindedBeaconBlockBellatrixJson `json:"data"`
}

type capellaProduceBlindedBlockResponseJson struct {
	Version string                         `json:"version" enum:"true"`
	Data    *BlindedBeaconBlockCapellaJson `json:"data"`
}

type denebProduceBlindedBlockResponseJson struct {
	Version string                               `json:"version" enum:"true"`
	Data    *BlindedBeaconBlockContentsDenebJson `json:"data"`
}

func serializeProducedV2Block(response interface{}) (apimiddleware.RunDefault, []byte, apimiddleware.ErrorJson) {
	respContainer, ok := response.(*ProduceBlockResponseV2Json)
	if !ok {
		return false, nil, apimiddleware.InternalServerError(errors.New("container is not of the correct type"))
	}

	var actualRespContainer interface{}
	switch {
	case strings.EqualFold(respContainer.Version, strings.ToLower(ethpbv2.Version_PHASE0.String())):
		actualRespContainer = &phase0ProduceBlockResponseJson{
			Version: respContainer.Version,
			Data:    respContainer.Data.Phase0Block,
		}
	case strings.EqualFold(respContainer.Version, strings.ToLower(ethpbv2.Version_ALTAIR.String())):
		actualRespContainer = &altairProduceBlockResponseJson{
			Version: respContainer.Version,
			Data:    respContainer.Data.AltairBlock,
		}
	case strings.EqualFold(respContainer.Version, strings.ToLower(ethpbv2.Version_BELLATRIX.String())):
		actualRespContainer = &bellatrixProduceBlockResponseJson{
			Version: respContainer.Version,
			Data:    respContainer.Data.BellatrixBlock,
		}
	case strings.EqualFold(respContainer.Version, strings.ToLower(ethpbv2.Version_CAPELLA.String())):
		actualRespContainer = &capellaProduceBlockResponseJson{
			Version: respContainer.Version,
			Data:    respContainer.Data.CapellaBlock,
		}
	case strings.EqualFold(respContainer.Version, strings.ToLower(ethpbv2.Version_DENEB.String())):
		actualRespContainer = &denebProduceBlockResponseJson{
			Version: respContainer.Version,
			Data:    respContainer.Data.DenebContents,
		}
	default:
		return false, nil, apimiddleware.InternalServerError(fmt.Errorf("unsupported block version '%s'", respContainer.Version))
	}

	j, err := json.Marshal(actualRespContainer)
	if err != nil {
		return false, nil, apimiddleware.InternalServerErrorWithMessage(err, "could not marshal response")
	}
	return false, j, nil
}

func serializeProducedBlindedBlock(response interface{}) (apimiddleware.RunDefault, []byte, apimiddleware.ErrorJson) {
	respContainer, ok := response.(*ProduceBlindedBlockResponseJson)
	if !ok {
		return false, nil, apimiddleware.InternalServerError(errors.New("container is not of the correct type"))
	}

	var actualRespContainer interface{}
	switch {
	case strings.EqualFold(respContainer.Version, strings.ToLower(ethpbv2.Version_PHASE0.String())):
		actualRespContainer = &phase0ProduceBlockResponseJson{
			Version: respContainer.Version,
			Data:    respContainer.Data.Phase0Block,
		}
	case strings.EqualFold(respContainer.Version, strings.ToLower(ethpbv2.Version_ALTAIR.String())):
		actualRespContainer = &altairProduceBlockResponseJson{
			Version: respContainer.Version,
			Data:    respContainer.Data.AltairBlock,
		}
	case strings.EqualFold(respContainer.Version, strings.ToLower(ethpbv2.Version_BELLATRIX.String())):
		actualRespContainer = &bellatrixProduceBlindedBlockResponseJson{
			Version: respContainer.Version,
			Data:    respContainer.Data.BellatrixBlock,
		}
	case strings.EqualFold(respContainer.Version, strings.ToLower(ethpbv2.Version_CAPELLA.String())):
		actualRespContainer = &capellaProduceBlindedBlockResponseJson{
			Version: respContainer.Version,
			Data:    respContainer.Data.CapellaBlock,
		}
	case strings.EqualFold(respContainer.Version, strings.ToLower(ethpbv2.Version_DENEB.String())):
		actualRespContainer = &denebProduceBlindedBlockResponseJson{
			Version: respContainer.Version,
			Data:    respContainer.Data.DenebContents,
		}
	default:
		return false, nil, apimiddleware.InternalServerError(fmt.Errorf("unsupported block version '%s'", respContainer.Version))
	}

	j, err := json.Marshal(actualRespContainer)
	if err != nil {
		return false, nil, apimiddleware.InternalServerErrorWithMessage(err, "could not marshal response")
	}
	return false, j, nil
}
