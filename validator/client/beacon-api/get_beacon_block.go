package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	neturl "net/url"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/validator"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

func (c beaconApiValidatorClient) getBeaconBlock(ctx context.Context, slot primitives.Slot, randaoReveal []byte, graffiti []byte) (*ethpb.GenericBeaconBlock, error) {
	queryParams := neturl.Values{}
	queryParams.Add("randao_reveal", hexutil.Encode(randaoReveal))

	if len(graffiti) > 0 {
		queryParams.Add("graffiti", hexutil.Encode(graffiti))
	}

	queryUrl := buildURL(fmt.Sprintf("/eth/v3/validator/blocks/%d", slot), queryParams)

	// Since we don't know yet what the json looks like, we unmarshal into an abstract structure that has only a version
	// and a blob of data
	produceBlockResponseJson := validator.ProduceBlockV3Response{}
	if _, err := c.jsonRestHandler.GetRestJsonResponse(ctx, queryUrl, &produceBlockResponseJson); err != nil {
		return nil, errors.Wrap(err, "failed to query GET REST endpoint")
	}

	// Once we know what the consensus version is, we can go ahead and unmarshal into the specific structs unique to each version
	decoder := json.NewDecoder(bytes.NewReader(produceBlockResponseJson.Data))
	decoder.DisallowUnknownFields()

	var response *ethpb.GenericBeaconBlock
	switch produceBlockResponseJson.Version {
	case "phase0":
		jsonPhase0Block := shared.BeaconBlock{}
		if err := decoder.Decode(&jsonPhase0Block); err != nil {
			return nil, errors.Wrap(err, "failed to decode phase0 block response json")
		}
		genericBlock, err := jsonPhase0Block.ToGeneric()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get phase0 block")
		}
		response = genericBlock
	case "altair":
		jsonAltairBlock := shared.BeaconBlockAltair{}
		if err := decoder.Decode(&jsonAltairBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode altair block response json")
		}
		genericBlock, err := jsonAltairBlock.ToGeneric()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get altair block")
		}
		response = genericBlock
	case "bellatrix":
		if produceBlockResponseJson.ExecutionPayloadBlinded {
			jsonBellatrixBlock := shared.BlindedBeaconBlockBellatrix{}
			if err := decoder.Decode(&jsonBellatrixBlock); err != nil {
				return nil, errors.Wrap(err, "failed to decode bellatrix block response json")
			}
			genericBlock, err := jsonBellatrixBlock.ToGeneric()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get bellatrix block")
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
	case "capella":
		if produceBlockResponseJson.ExecutionPayloadBlinded {
			jsonCapellaBlock := shared.BlindedBeaconBlockCapella{}
			if err := decoder.Decode(&jsonCapellaBlock); err != nil {
				return nil, errors.Wrap(err, "failed to decode capella block response json")
			}
			genericBlock, err := jsonCapellaBlock.ToGeneric()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get capella block")
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
	case "deneb":
		if produceBlockResponseJson.ExecutionPayloadBlinded {
			jsonDenebBlockContents := shared.BlindedBeaconBlockContentsDeneb{}
			if err := decoder.Decode(&jsonDenebBlockContents); err != nil {
				return nil, errors.Wrap(err, "failed to decode deneb block response json")
			}
			genericBlock, err := jsonDenebBlockContents.ToGeneric()
			if err != nil {
				return nil, errors.Wrap(err, "could not convert deneb block contents to generic block")
			}
			response = genericBlock
		} else {
			jsonDenebBlockContents := shared.BeaconBlockContentsDeneb{}
			if err := decoder.Decode(&jsonDenebBlockContents); err != nil {
				return nil, errors.Wrap(err, "failed to decode deneb block response json")
			}
			genericBlock, err := jsonDenebBlockContents.ToGeneric()
			if err != nil {
				return nil, errors.Wrap(err, "could not convert deneb block contents to generic block")
			}
			response = genericBlock
		}
	default:
		return nil, errors.Errorf("unsupported consensus version `%s`", produceBlockResponseJson.Version)
	}
	return response, nil
}
