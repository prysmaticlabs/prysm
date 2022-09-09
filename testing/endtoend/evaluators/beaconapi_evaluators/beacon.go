package beaconapi_evaluators

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"google.golang.org/grpc"
)

// GET "/eth/v1/beacon/blocks/{block_id}"
// GET "/eth/v1/beacon/blocks/{block_id}/root"
func withCompareBeaconBlocks(beaconNodeIdx int, conn *grpc.ClientConn) error {
	ctx := context.Background()
	beaconClient := service.NewBeaconChainClient(conn)
	genesisData, err := beaconClient.GetGenesis(ctx, &empty.Empty{})
	if err != nil {
		return err
	}
	currentEpoch := slots.EpochsSinceGenesis(genesisData.Data.GenesisTime.AsTime())
	respJSONPrysm := &apimiddleware.BlockResponseJson{}
	respJSONLighthouse := &apimiddleware.BlockResponseJson{}
	var check string
	if currentEpoch < 3 {
		check = "genesis"
	} else {
		check = "finalized"
	}
	if currentEpoch < params.BeaconConfig().AltairForkEpoch {
		resp, err := beaconClient.GetBlock(ctx, &ethpbv1.BlockRequest{
			BlockId: []byte("head"),
		})
		if err != nil {
			return err
		}

		fmt.Printf("version: 1 current Epoch: %d", currentEpoch)
		if err := doMiddlewareJSONGetRequest(
			v1MiddlewarePathTemplate,
			"/beacon/blocks/head",
			beaconNodeIdx,
			respJSONPrysm,
		); err != nil {
			return err
		}

		if err := doMiddlewareJSONGetRequest(
			v1MiddlewarePathTemplate,
			"/beacon/blocks/head",
			beaconNodeIdx,
			respJSONLighthouse,
			"lighthouse",
		); err != nil {
			return err
		}

		if hexutil.Encode(resp.Data.Signature) != respJSONPrysm.Data.Signature {
			return fmt.Errorf("API Middleware block signature  %s does not match gRPC block signature %s",
				respJSONPrysm.Data.Signature,
				hexutil.Encode(resp.Data.Signature))
		}

		if !reflect.DeepEqual(respJSONPrysm, respJSONLighthouse) {
			p, err := json.Marshal(respJSONPrysm)
			if err != nil {
				return err
			}
			l, err := json.Marshal(respJSONLighthouse)
			if err != nil {
				return err
			}
			return fmt.Errorf("prysm response %s does not match lighthouse response %s",
				string(p),
				string(l))
		}

		blockP := &ethpb.SignedBeaconBlock{}
		blockL := &ethpb.SignedBeaconBlock{}

		sszrspL, err := doMiddlewareSSZGetRequest(
			v1MiddlewarePathTemplate,
			"/beacon/blocks/"+check,
			beaconNodeIdx,
			"lighthouse",
		)
		if err != nil {
			return err
		}

		sszrspP, err := doMiddlewareSSZGetRequest(
			v1MiddlewarePathTemplate,
			"/beacon/blocks/"+check,
			beaconNodeIdx,
		)
		if err != nil {
			return err
		}
		if err := blockP.UnmarshalSSZ(sszrspP); err != nil {
			return err
		}
		if err := blockL.UnmarshalSSZ(sszrspL); err != nil {
			return err
		}
		if len(blockP.Signature) == 0 || len(blockL.Signature) == 0 || hexutil.Encode(blockP.Signature) != hexutil.Encode(blockL.Signature) {
			return fmt.Errorf("prysm response %v does not match lighthouse response %v",
				blockP,
				blockL)
		}
		//if !bytes.Equal(sszrspL, sszrspP) {
		//	return fmt.Errorf("prysm ssz response %s does not match lighthouse ssz response %s",
		//		hexutil.Encode(sszrspP),
		//		hexutil.Encode(sszrspL))
		//}

	} else {
		resp, err := beaconClient.GetBlockV2(ctx, &ethpbv2.BlockRequestV2{
			BlockId: []byte("head"),
		})
		if err != nil {
			return err
		}
		fmt.Printf("version: 2 current Epoch: %d", currentEpoch)
		if err := doMiddlewareJSONGetRequest(
			v2MiddlewarePathTemplate,
			"/beacon/blocks/head",
			beaconNodeIdx,
			respJSONPrysm,
		); err != nil {
			return err
		}

		if err := doMiddlewareJSONGetRequest(
			v2MiddlewarePathTemplate,
			"/beacon/blocks/head",
			beaconNodeIdx,
			respJSONLighthouse,
			"lighthouse",
		); err != nil {
			return err
		}

		if hexutil.Encode(resp.Data.Signature) != respJSONPrysm.Data.Signature {
			return fmt.Errorf("API Middleware block signature  %s does not match gRPC block signature %s",
				respJSONPrysm.Data.Signature,
				hexutil.Encode(resp.Data.Signature))
		}

		if !reflect.DeepEqual(respJSONPrysm, respJSONLighthouse) {
			p, err := json.Marshal(respJSONPrysm)
			if err != nil {
				return err
			}
			l, err := json.Marshal(respJSONLighthouse)
			if err != nil {
				return err
			}
			return fmt.Errorf("prysm response %s does not match lighthouse response %s",
				string(p),
				string(l))
		}

		blockP := &ethpb.SignedBeaconBlock{}
		blockL := &ethpb.SignedBeaconBlock{}

		sszrspL, err := doMiddlewareSSZGetRequest(
			v2MiddlewarePathTemplate,
			"/beacon/blocks/"+check,
			beaconNodeIdx,
			"lighthouse",
		)
		if err != nil {
			return err
		}

		sszrspP, err := doMiddlewareSSZGetRequest(
			v2MiddlewarePathTemplate,
			"/beacon/blocks/"+check,
			beaconNodeIdx,
		)
		if err != nil {
			return err
		}
		if err := blockP.UnmarshalSSZ(sszrspP); err != nil {
			return err
		}
		if err := blockL.UnmarshalSSZ(sszrspL); err != nil {
			return err
		}
		if len(blockP.Signature) == 0 || len(blockL.Signature) == 0 || hexutil.Encode(blockP.Signature) != hexutil.Encode(blockL.Signature) {
			return fmt.Errorf("prysm response %v does not match lighthouse response %v",
				blockP,
				blockL)
		}
		//if !bytes.Equal(sszrspL, sszrspP) {
		//	return fmt.Errorf("prysm ssz response %s does not match lighthouse ssz response %s",
		//		hexutil.Encode(sszrspP),
		//		hexutil.Encode(sszrspL))
		//}
	}

	blockroot, err := beaconClient.GetBlockRoot(ctx, &ethpbv1.BlockRequest{
		BlockId: []byte("head"),
	})
	if err != nil {
		return err
	}
	blockrootJSON := &apimiddleware.BlockRootResponseJson{}
	if err := doMiddlewareJSONGetRequest(
		v1MiddlewarePathTemplate,
		"/beacon/blocks/head/root",
		beaconNodeIdx,
		blockrootJSON,
	); err != nil {
		return err
	}
	if hexutil.Encode(blockroot.Data.Root) != blockrootJSON.Data.Root {
		return fmt.Errorf("API Middleware block root  %s does not match gRPC block root %s",
			blockrootJSON.Data.Root,
			hexutil.Encode(blockroot.Data.Root))
	}
	return nil
}
