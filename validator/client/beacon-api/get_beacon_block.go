package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/validator"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
)

type abstractProduceBlockResponseJson struct {
	Version string          `json:"version"`
	Data    json.RawMessage `json:"data"`
}

func (c beaconApiValidatorClient) getBeaconBlock(ctx context.Context, slot primitives.Slot, randaoReveal []byte, graffiti []byte) (*ethpb.GenericBeaconBlock, error) {
	queryParams := neturl.Values{}
	queryParams.Add("randao_reveal", hexutil.Encode(randaoReveal))
	if len(graffiti) > 0 {
		queryParams.Add("graffiti", hexutil.Encode(graffiti))
	}

	var ver string
	var blinded bool
	var decoder *json.Decoder

	// Try v3 endpoint first. If it's not supported, then we fall back to older endpoints.
	// We try the blinded block endpoint first. If it fails, we assume that we got a full block and try the full block endpoint.
	queryUrl := buildURL(fmt.Sprintf("/eth/v3/validator/blocks/%d", slot), queryParams)
	produceBlockV3ResponseJson := validator.ProduceBlockV3Response{}
	errJson, err := c.jsonRestHandler.GetRestJsonResponse(ctx, queryUrl, &produceBlockV3ResponseJson)
	if err == nil {
		ver = produceBlockV3ResponseJson.Version
		blinded = produceBlockV3ResponseJson.ExecutionPayloadBlinded
		decoder = json.NewDecoder(bytes.NewReader(produceBlockV3ResponseJson.Data))
	} else if errJson == nil {
		return nil, errors.Wrap(err, "failed to query GET REST endpoint")
	} else if errJson.Code == http.StatusNotFound {
		log.Debug("Endpoint /eth/v3/validator/blocks is not supported, falling back to older endpoints for block proposal.")
		produceBlindedBlockResponseJson := abstractProduceBlockResponseJson{}
		queryUrl = buildURL(fmt.Sprintf("/eth/v1/validator/blinded_blocks/%d", slot), queryParams)
		errJson, err = c.jsonRestHandler.GetRestJsonResponse(ctx, queryUrl, &produceBlindedBlockResponseJson)
		if err == nil {
			ver = produceBlindedBlockResponseJson.Version
			blinded = true
			decoder = json.NewDecoder(bytes.NewReader(produceBlindedBlockResponseJson.Data))
		} else {
			log.Debug("Endpoint /eth/v1/validator/blinded_blocks failed to produce a blinded block, trying /eth/v2/validator/blocks.")
			produceBlockResponseJson := abstractProduceBlockResponseJson{}
			queryUrl = buildURL(fmt.Sprintf("/eth/v2/validator/blocks/%d", slot), queryParams)
			errJson, err = c.jsonRestHandler.GetRestJsonResponse(ctx, queryUrl, &produceBlockResponseJson)
			if err != nil {
				return nil, errors.Wrap(err, "failed to query GET REST endpoint")
			}
			ver = produceBlockResponseJson.Version
			blinded = false
			decoder = json.NewDecoder(bytes.NewReader(produceBlockResponseJson.Data))
		}
	} else {
		return nil, fmt.Errorf("failed to query GET REST endpoint: %s (status code %d)", errJson.Message, errJson.Code)
	}

	var response *ethpb.GenericBeaconBlock
	switch ver {
	case version.String(version.Phase0):
		jsonPhase0Block := shared.BeaconBlock{}
		if err := decoder.Decode(&jsonPhase0Block); err != nil {
			return nil, errors.Wrap(err, "failed to decode phase0 block response json")
		}
		genericBlock, err := jsonPhase0Block.ToGeneric()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get phase0 block")
		}
		response = genericBlock
	case version.String(version.Altair):
		jsonAltairBlock := shared.BeaconBlockAltair{}
		if err := decoder.Decode(&jsonAltairBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode altair block response json")
		}
		genericBlock, err := jsonAltairBlock.ToGeneric()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get altair block")
		}
		response = genericBlock
	case version.String(version.Bellatrix):
		if blinded {
			jsonBellatrixBlock := shared.BlindedBeaconBlockBellatrix{}
			if err := decoder.Decode(&jsonBellatrixBlock); err != nil {
				return nil, errors.Wrap(err, "failed to decode blinded bellatrix block response json")
			}
			genericBlock, err := jsonBellatrixBlock.ToGeneric()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get blinded bellatrix block")
			}
			response = genericBlock
		} else {
			jsonBellatrixBlock := shared.BeaconBlockBellatrix{}
			if err := decoder.Decode(&jsonBellatrixBlock); err != nil {
				return nil, errors.Wrap(err, "failed to decode bellatrix block response json")
			}
			genericBlock, err := jsonBellatrixBlock.ToGeneric()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get bellatrix block")
			}
			response = genericBlock
		}
	case version.String(version.Capella):
		if blinded {
			jsonCapellaBlock := shared.BlindedBeaconBlockCapella{}
			if err := decoder.Decode(&jsonCapellaBlock); err != nil {
				return nil, errors.Wrap(err, "failed to decode blinded capella block response json")
			}
			genericBlock, err := jsonCapellaBlock.ToGeneric()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get blinded capella block")
			}
			response = genericBlock
		} else {
			jsonCapellaBlock := shared.BeaconBlockCapella{}
			if err := decoder.Decode(&jsonCapellaBlock); err != nil {
				return nil, errors.Wrap(err, "failed to decode capella block response json")
			}
			genericBlock, err := jsonCapellaBlock.ToGeneric()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get capella block")
			}
			response = genericBlock
		}
	case version.String(version.Deneb):
		if blinded {
			jsonDenebBlockContents := shared.BlindedBeaconBlockContentsDeneb{}
			if err := decoder.Decode(&jsonDenebBlockContents); err != nil {
				return nil, errors.Wrap(err, "failed to decode blinded deneb block response json")
			}
			genericBlock, err := jsonDenebBlockContents.ToGeneric()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get blinded deneb block")
			}
			response = genericBlock
		} else {
			jsonDenebBlockContents := shared.BeaconBlockContentsDeneb{}
			if err := decoder.Decode(&jsonDenebBlockContents); err != nil {
				return nil, errors.Wrap(err, "failed to decode deneb block response json")
			}
			genericBlock, err := jsonDenebBlockContents.ToGeneric()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get deneb block")
			}
			response = genericBlock
		}
	default:
		return nil, errors.Errorf("unsupported consensus version `%s`", ver)
	}
	return response, nil
}
