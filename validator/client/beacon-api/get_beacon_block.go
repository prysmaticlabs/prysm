package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	neturl "net/url"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type abstractProduceBlockResponseJson struct {
	Version string          `json:"version" enum:"true"`
	Data    json.RawMessage `json:"data"`
}

func (c beaconApiValidatorClient) getBeaconBlock(ctx context.Context, slot primitives.Slot, randaoReveal []byte, graffiti []byte) (*ethpb.GenericBeaconBlock, error) {
	queryParams := neturl.Values{}
	queryParams.Add("randao_reveal", hexutil.Encode(randaoReveal))

	if len(graffiti) > 0 {
		queryParams.Add("graffiti", hexutil.Encode(graffiti))
	}

	queryUrl := buildURL(fmt.Sprintf("/eth/v2/validator/blocks/%d", slot), queryParams)

	// Since we don't know yet what the json looks like, we unmarshal into an abstract structure that has only a version
	// and a blob of data
	produceBlockResponseJson := abstractProduceBlockResponseJson{}
	if _, err := c.jsonRestHandler.GetRestJsonResponse(ctx, queryUrl, &produceBlockResponseJson); err != nil {
		return nil, errors.Wrap(err, "failed to query GET REST endpoint")
	}

	// Once we know what the consensus version is, we can go ahead and unmarshal into the specific structs unique to each version
	decoder := json.NewDecoder(bytes.NewReader(produceBlockResponseJson.Data))

	response := &ethpb.GenericBeaconBlock{}

	switch produceBlockResponseJson.Version {
	case "phase0":
		jsonPhase0Block := apimiddleware.BeaconBlockJson{}
		if err := decoder.Decode(&jsonPhase0Block); err != nil {
			return nil, errors.Wrap(err, "failed to decode phase0 block response json")
		}

		phase0Block, err := c.beaconBlockConverter.ConvertRESTPhase0BlockToProto(&jsonPhase0Block)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get phase0 block")
		}
		response.Block = &ethpb.GenericBeaconBlock_Phase0{
			Phase0: phase0Block,
		}

	case "altair":
		jsonAltairBlock := apimiddleware.BeaconBlockAltairJson{}
		if err := decoder.Decode(&jsonAltairBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode altair block response json")
		}

		altairBlock, err := c.beaconBlockConverter.ConvertRESTAltairBlockToProto(&jsonAltairBlock)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get altair block")
		}
		response.Block = &ethpb.GenericBeaconBlock_Altair{
			Altair: altairBlock,
		}

	case "bellatrix":
		jsonBellatrixBlock := apimiddleware.BeaconBlockBellatrixJson{}
		if err := decoder.Decode(&jsonBellatrixBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode bellatrix block response json")
		}

		bellatrixBlock, err := c.beaconBlockConverter.ConvertRESTBellatrixBlockToProto(&jsonBellatrixBlock)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get bellatrix block")
		}
		response.Block = &ethpb.GenericBeaconBlock_Bellatrix{
			Bellatrix: bellatrixBlock,
		}

	case "capella":
		jsonCapellaBlock := apimiddleware.BeaconBlockCapellaJson{}
		if err := decoder.Decode(&jsonCapellaBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode capella block response json")
		}

		capellaBlock, err := c.beaconBlockConverter.ConvertRESTCapellaBlockToProto(&jsonCapellaBlock)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get capella block")
		}
		response.Block = &ethpb.GenericBeaconBlock_Capella{
			Capella: capellaBlock,
		}
	case "deneb":
		jsonDenebBlockContents := shared.BeaconBlockContentsDeneb{}
		if err := decoder.Decode(&jsonDenebBlockContents); err != nil {
			return nil, errors.Wrap(err, "failed to decode deneb block response json")
		}
		genericBlock, err := jsonDenebBlockContents.ToGeneric()
		if err != nil {
			return nil, errors.Wrap(err, "could not convert deneb block contents to generic block")
		}
		response = genericBlock
	default:
		return nil, errors.Errorf("unsupported consensus version `%s`", produceBlockResponseJson.Version)
	}
	return response, nil
}
