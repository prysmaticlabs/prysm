package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"google.golang.org/grpc"
)

type abstractSignedBlindedBlockResponseJson struct {
	Version             string          `json:"version" enum:"true"`
	ExecutionOptimistic bool            `json:"execution_optimistic"`
	Finalized           bool            `json:"finalized"`
	Data                json.RawMessage `json:"data"`
}

type streamBlocksAltairClient struct {
	grpc.ClientStream
	ctx                 context.Context
	beaconApiClient     beaconApiValidatorClient
	streamBlocksRequest *ethpb.StreamBlocksRequest
	prevBlockSlot       primitives.Slot
	pingDelay           time.Duration
}

type headSignedBeaconBlockResult struct {
	streamBlocksResponse *ethpb.StreamBlocksResponse
	executionOptimistic  bool
	slot                 primitives.Slot
}

func (c beaconApiValidatorClient) streamBlocks(ctx context.Context, in *ethpb.StreamBlocksRequest, pingDelay time.Duration) ethpb.BeaconNodeValidator_StreamBlocksAltairClient {
	return &streamBlocksAltairClient{
		ctx:                 ctx,
		beaconApiClient:     c,
		streamBlocksRequest: in,
		pingDelay:           pingDelay,
	}
}

func (c *streamBlocksAltairClient) Recv() (*ethpb.StreamBlocksResponse, error) {
	result, err := c.beaconApiClient.getHeadSignedBeaconBlock(c.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest signed block")
	}

	// We keep querying the beacon chain for the latest block until we receive a new slot
	for (c.streamBlocksRequest.VerifiedOnly && result.executionOptimistic) || c.prevBlockSlot == result.slot {
		select {
		case <-time.After(c.pingDelay):
			result, err = c.beaconApiClient.getHeadSignedBeaconBlock(c.ctx)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get latest signed block")
			}
		case <-c.ctx.Done():
			return nil, errors.New("context canceled")
		}
	}

	c.prevBlockSlot = result.slot
	return result.streamBlocksResponse, nil
}

func (c beaconApiValidatorClient) getHeadSignedBeaconBlock(ctx context.Context) (*headSignedBeaconBlockResult, error) {
	// Since we don't know yet what the json looks like, we unmarshal into an abstract structure that has only a version
	// and a blob of data
	signedBlockResponseJson := abstractSignedBlindedBlockResponseJson{}
	errJson, err := c.jsonRestHandler.Get(ctx, "/eth/v1/beacon/blinded_blocks/head", &signedBlockResponseJson)
	if err != nil {
		return nil, errors.Wrapf(err, msgUnexpectedError)
	}
	if errJson != nil {
		return nil, errJson
	}

	// Once we know what the consensus version is, we can go ahead and unmarshal into the specific structs unique to each version
	decoder := json.NewDecoder(bytes.NewReader(signedBlockResponseJson.Data))

	response := &ethpb.StreamBlocksResponse{}
	var slot primitives.Slot

	switch signedBlockResponseJson.Version {
	case "phase0":
		jsonPhase0Block := shared.SignedBeaconBlock{}
		if err := decoder.Decode(&jsonPhase0Block); err != nil {
			return nil, errors.Wrap(err, "failed to decode signed phase0 block response json")
		}
		phase0Block, err := jsonPhase0Block.ToConsensus()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get signed phase0 block")
		}
		response.Block = &ethpb.StreamBlocksResponse_Phase0Block{
			Phase0Block: &ethpb.SignedBeaconBlock{
				Signature: phase0Block.Signature,
				Block:     &ethpb.BeaconBlock{Slot: phase0Block.Block.Slot},
			},
		}

		slot = phase0Block.Block.Slot

	case "altair":
		jsonAltairBlock := shared.SignedBeaconBlockAltair{}
		if err := decoder.Decode(&jsonAltairBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode signed altair block response json")
		}
		altairBlock, err := jsonAltairBlock.ToConsensus()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get signed altair block")
		}

		response.Block = &ethpb.StreamBlocksResponse_AltairBlock{
			AltairBlock: &ethpb.SignedBeaconBlockAltair{
				Signature: altairBlock.Signature,
				Block:     &ethpb.BeaconBlockAltair{Slot: altairBlock.Block.Slot},
			},
		}

		slot = altairBlock.Block.Slot

	case "bellatrix":
		jsonBellatrixBlock := shared.SignedBlindedBeaconBlockBellatrix{}
		if err := decoder.Decode(&jsonBellatrixBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode signed bellatrix block response json")
		}
		bellatrixBlock, err := jsonBellatrixBlock.ToConsensus()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get signed bellatrix block")
		}

		response.Block = &ethpb.StreamBlocksResponse_BellatrixBlock{
			BellatrixBlock: &ethpb.SignedBeaconBlockBellatrix{
				Signature: bellatrixBlock.Signature,
				Block:     &ethpb.BeaconBlockBellatrix{Slot: bellatrixBlock.Block.Slot},
			},
		}

		slot = bellatrixBlock.Block.Slot

	case "capella":
		jsonCapellaBlock := shared.SignedBlindedBeaconBlockCapella{}
		if err := decoder.Decode(&jsonCapellaBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode signed capella block response json")
		}
		capellaBlock, err := jsonCapellaBlock.ToConsensus()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get signed capella block")
		}

		response.Block = &ethpb.StreamBlocksResponse_CapellaBlock{
			CapellaBlock: &ethpb.SignedBeaconBlockCapella{
				Signature: capellaBlock.Signature,
				Block:     &ethpb.BeaconBlockCapella{Slot: capellaBlock.Block.Slot},
			},
		}

		slot = capellaBlock.Block.Slot

	case "deneb":
		jsonDenebBlock := shared.SignedBlindedBeaconBlockDeneb{}
		if err := decoder.Decode(&jsonDenebBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode signed deneb block response json")
		}
		denebBlock, err := jsonDenebBlock.ToConsensus()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get signed deneb block")
		}

		response.Block = &ethpb.StreamBlocksResponse_DenebBlock{
			DenebBlock: &ethpb.SignedBeaconBlockDeneb{
				Signature: denebBlock.Signature,
				Block:     &ethpb.BeaconBlockDeneb{Slot: denebBlock.Message.Slot},
			},
		}

		slot = denebBlock.Message.Slot

	default:
		return nil, errors.Errorf("unsupported block version `%s`", signedBlockResponseJson.Version)
	}

	return &headSignedBeaconBlockResult{
		streamBlocksResponse: response,
		executionOptimistic:  signedBlockResponseJson.ExecutionOptimistic,
		slot:                 slot,
	}, nil
}
