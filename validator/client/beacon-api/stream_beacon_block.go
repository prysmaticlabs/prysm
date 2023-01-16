package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/grpc"
)

type streamBeaconBlockClient struct {
	grpc.ClientStream
	ctx             context.Context
	verifiedOnly    bool
	lastRecvSlot    types.Slot
	jsonRestHandler jsonRestHandler
}

func (c *beaconApiValidatorClient) streamBlocksAltair(ctx context.Context, verifiedOnly bool) ethpb.BeaconNodeValidator_StreamBlocksAltairClient {
	return &streamBeaconBlockClient{
		ctx:          ctx,
		verifiedOnly: verifiedOnly,
	}
}

type abstractGetBlockResponseJson struct {
	Version             string          `json:"version" enum:"true"`
	ExecutionOptimistic bool            `json:"execution_optimistic"`
	Data                json.RawMessage `json:"data"`
}

type phase0BlockJson struct {
	apimiddleware.BeaconBlockJson
	Signature string `json:"signature" hex:"true"`
}

type altairBlockJson struct {
	apimiddleware.BeaconBlockAltairJson
	Signature string `json:"signature" hex:"true"`
}

type bellatrixBlockJson struct {
	apimiddleware.BeaconBlockBellatrixJson
	Signature string `json:"signature" hex:"true"`
}

type capellaBlockJson struct {
	apimiddleware.BeaconBlockCapellaJson
	Signature string `json:"signature" hex:"true"`
}

func (c *streamBeaconBlockClient) Recv() (*ethpb.StreamBlocksResponse, error) {
	return c.getBlock()
}

func (c *streamBeaconBlockClient) getBlock() (*ethpb.StreamBlocksResponse, error) {
	getBlockResponseJson := abstractGetBlockResponseJson{}
	if _, err := c.jsonRestHandler.GetRestJsonResponse("/eth/v2/beacon/blocks/head", &getBlockResponseJson); err != nil {
		return nil, errors.Wrap(err, "failed to query GET REST endpoint")
	}

	// Return early if we only want verified blocks but the block that we just received is optimistic
	if c.verifiedOnly && getBlockResponseJson.ExecutionOptimistic {
		return nil, nil
	}

	// Once we know what the consensus version is, we can go ahead and unmarshal into the specific structs unique to each version
	decoder := json.NewDecoder(bytes.NewReader(getBlockResponseJson.Data))
	decoder.DisallowUnknownFields()

	response := &ethpb.StreamBlocksResponse{}

	switch getBlockResponseJson.Version {
	case "phase0":
		jsonPhase0Block := phase0BlockJson{}
		if err := decoder.Decode(&jsonPhase0Block); err != nil {
			return nil, errors.Wrap(err, "failed to decode phase0 block response json")
		}

		phase0Block, err := convertRESTPhase0BlockToProto(&jsonPhase0Block.BeaconBlockJson)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get phase0 block")
		}

		signature, err := hexutil.Decode(jsonPhase0Block.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode signature `%s`", jsonPhase0Block.Signature)
		}

		response.Block = &ethpb.StreamBlocksResponse_Phase0Block{
			Phase0Block: &ethpb.SignedBeaconBlock{
				Block:     phase0Block,
				Signature: signature,
			},
		}

	case "altair":
		jsonAltairBlock := altairBlockJson{}
		if err := decoder.Decode(&jsonAltairBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode altair block response json")
		}

		altairBlock, err := convertRESTAltairBlockToProto(&jsonAltairBlock.BeaconBlockAltairJson)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get altair block")
		}

		signature, err := hexutil.Decode(jsonAltairBlock.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode signature `%s`", jsonAltairBlock.Signature)
		}

		response.Block = &ethpb.StreamBlocksResponse_AltairBlock{
			AltairBlock: &ethpb.SignedBeaconBlockAltair{
				Block:     altairBlock,
				Signature: signature,
			},
		}

	case "bellatrix":
		jsonBellatrixBlock := bellatrixBlockJson{}
		if err := decoder.Decode(&jsonBellatrixBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode bellatrix block response json")
		}

		bellatrixBlock, err := convertRESTBellatrixBlockToProto(&jsonBellatrixBlock.BeaconBlockBellatrixJson)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get bellatrix block")
		}

		signature, err := hexutil.Decode(jsonBellatrixBlock.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode signature `%s`", jsonBellatrixBlock.Signature)
		}

		response.Block = &ethpb.StreamBlocksResponse_BellatrixBlock{
			BellatrixBlock: &ethpb.SignedBeaconBlockBellatrix{
				Block:     bellatrixBlock,
				Signature: signature,
			},
		}

	default:
		return nil, errors.Errorf("unsupported consensus version `%s`", getBlockResponseJson.Version)
	}

	return response, nil
}
