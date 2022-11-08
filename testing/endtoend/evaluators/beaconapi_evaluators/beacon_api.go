package beaconapi_evaluators

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	"github.com/prysmaticlabs/prysm/v3/time/slots"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/grpc"
)

func withCompareBeaconAPIs(beaconNodeIdx int, conn *grpc.ClientConn) error {
	ctx := context.Background()
	beaconClient := service.NewBeaconChainClient(conn)
	genesisData, err := beaconClient.GetGenesis(ctx, &empty.Empty{})
	if err != nil {
		return errors.Wrap(err, "error getting genesis data")
	}
	currentEpoch := slots.EpochsSinceGenesis(genesisData.Data.GenesisTime.AsTime())
	type metadata struct {
		basepath        string
		params          func(encoding string, currentEpoch types.Epoch) []string
		prysmResps      map[string]interface{}
		lighthouseResps map[string]interface{}
	}
	beaconPathsAndObjects := map[string]metadata{
		"/beacon/genesis": {
			basepath: v1MiddlewarePathTemplate,
			params: func(_ string, _ types.Epoch) []string {
				return []string{"skip"}
			},
			prysmResps: map[string]interface{}{
				"json": &apimiddleware.GenesisResponseJson{},
			},
			lighthouseResps: map[string]interface{}{
				"json": &apimiddleware.GenesisResponseJson{},
			},
		},
		"/beacon/states/{param1}/root": {
			basepath: v1MiddlewarePathTemplate,
			params: func(_ string, _ types.Epoch) []string {
				return []string{"head"}
			},
			prysmResps: map[string]interface{}{
				"json": &apimiddleware.StateRootResponseJson{},
			},
			lighthouseResps: map[string]interface{}{
				"json": &apimiddleware.StateRootResponseJson{},
			},
		},
		"/beacon/states/{param1}/finality_checkpoints": {
			basepath: v1MiddlewarePathTemplate,
			params: func(_ string, _ types.Epoch) []string {
				return []string{"head"}
			},
			prysmResps: map[string]interface{}{
				"json": &apimiddleware.StateFinalityCheckpointResponseJson{},
			},
			lighthouseResps: map[string]interface{}{
				"json": &apimiddleware.StateFinalityCheckpointResponseJson{},
			},
		},
		"/beacon/blocks/{param1}": {
			basepath: v2MiddlewarePathTemplate,
			params: func(t string, e types.Epoch) []string {
				if t == "ssz" {
					var check string
					if e < 4 {
						check = "genesis"
					} else {
						check = "finalized"
					}
					return []string{check}
				} else {
					return []string{"head"}
				}
			},
			prysmResps: map[string]interface{}{
				"json": &apimiddleware.BlockResponseJson{},
				"ssz":  []byte{},
			},
			lighthouseResps: map[string]interface{}{
				"json": &apimiddleware.BlockResponseJson{},
				"ssz":  []byte{},
			},
		},
		"/beacon/states/{param1}/fork": {
			basepath: v1MiddlewarePathTemplate,
			params: func(_ string, _ types.Epoch) []string {
				return []string{"finalized"}
			},
			prysmResps: map[string]interface{}{
				"json": &apimiddleware.StateForkResponseJson{},
			},
			lighthouseResps: map[string]interface{}{
				"json": &apimiddleware.StateForkResponseJson{},
			},
		},
		"/debug/beacon/states/{param1}": {
			basepath: v2MiddlewarePathTemplate,
			params: func(_ string, e types.Epoch) []string {
				return []string{"head"}
			},
			prysmResps: map[string]interface{}{
				"json": &apimiddleware.BeaconStateV2ResponseJson{},
			},
			lighthouseResps: map[string]interface{}{
				"json": &apimiddleware.BeaconStateV2ResponseJson{},
			},
		},
		//"/validator/duties/proposer/{param1}": {
		//	basepath: v1MiddlewarePathTemplate,
		//	params: func(_ string, e types.Epoch) []string {
		//		return []string{fmt.Sprintf("%v", e)}
		//	},
		//	prysmResps: map[string]interface{}{
		//		"json": &apimiddleware.ProposerDutiesResponseJson{},
		//	},
		//	lighthouseResps: map[string]interface{}{
		//		"json": &apimiddleware.ProposerDutiesResponseJson{},
		//	},
		//},
	}
	for path, meta := range beaconPathsAndObjects {
		for key, _ := range meta.prysmResps {
			switch key {
			case "json":
				params := meta.params("json", currentEpoch)
				if len(params) == 0 {
					continue
				}
				apipath := pathFromParams(path, params)
				fmt.Printf("json api path: %s", apipath)
				if err := compareJSONMulticlient(beaconNodeIdx, meta.basepath, apipath, beaconPathsAndObjects[path].prysmResps[key], beaconPathsAndObjects[path].lighthouseResps[key]); err != nil {
					return err
				}
				fmt.Printf("prysm ob: %v/n", beaconPathsAndObjects[path].prysmResps[key])
				fmt.Printf("lighthouse ob: %v", beaconPathsAndObjects[path].prysmResps[key])
			case "ssz":
				params := meta.params("ssz", currentEpoch)
				if len(params) == 0 {
					continue
				}
				apipath := pathFromParams(path, params)
				fmt.Printf("ssz api path: %s", apipath)
				prysmr, lighthouser, err := compareSSZMulticlient(beaconNodeIdx, meta.basepath, apipath)
				if err != nil {
					return err
				}
				beaconPathsAndObjects[path].prysmResps[key] = prysmr
				beaconPathsAndObjects[path].lighthouseResps[key] = lighthouser
			default:
				return fmt.Errorf("unknown encoding type %s", key)
			}
		}
	}
	forkPathData := beaconPathsAndObjects["/beacon/states/{param1}/fork"]
	fmt.Printf("forkdata %v, %v", forkPathData, forkPathData.prysmResps["json"])
	prysmForkData, ok := forkPathData.prysmResps["json"].(*apimiddleware.StateForkResponseJson)
	if !ok {
		return errors.New("failed to cast type")
	}
	lighthouseForkData, ok := forkPathData.lighthouseResps["json"].(*apimiddleware.StateForkResponseJson)
	if !ok {
		return errors.New("failed to cast type")
	}
	if prysmForkData.Data.Epoch != lighthouseForkData.Data.Epoch {
		return fmt.Errorf("prysm response %v does not match lighthouse response %v",
			prysmForkData,
			lighthouseForkData)
	}

	finalizedEpoch, err := strconv.ParseUint(prysmForkData.Data.Epoch, 10, 64)
	if err != nil {
		return err
	}
	blockPathData := beaconPathsAndObjects["/beacon/blocks/{param1}"]
	sszrspL, ok := blockPathData.prysmResps["ssz"].([]byte)
	if !ok {
		return errors.New("failed to cast type")
	}
	sszrspP, ok := blockPathData.lighthouseResps["ssz"].([]byte)
	if !ok {
		return errors.New("failed to cast type")
	}
	if finalizedEpoch < 8 {
		blockP := &ethpb.SignedBeaconBlock{}
		blockL := &ethpb.SignedBeaconBlock{}
		if err := blockL.UnmarshalSSZ(sszrspL); err != nil {
			return errors.Wrap(err, "lighthouse ssz error")
		}
		if err := blockP.UnmarshalSSZ(sszrspP); err != nil {
			return errors.Wrap(err, "prysm ssz error")
		}
		if len(blockP.Signature) == 0 || len(blockL.Signature) == 0 || hexutil.Encode(blockP.Signature) != hexutil.Encode(blockL.Signature) {
			return fmt.Errorf("prysm response %v does not match lighthouse response %v",
				blockP,
				blockL)
		}
	} else if finalizedEpoch > 8 && finalizedEpoch < 10 {
		blockP := &ethpb.SignedBeaconBlockAltair{}
		blockL := &ethpb.SignedBeaconBlockAltair{}
		if err := blockL.UnmarshalSSZ(sszrspL); err != nil {
			return errors.Wrap(err, "lighthouse ssz error")
		}
		if err := blockP.UnmarshalSSZ(sszrspP); err != nil {
			return errors.Wrap(err, "prysm ssz error")
		}

		if len(blockP.Signature) == 0 || len(blockL.Signature) == 0 || hexutil.Encode(blockP.Signature) != hexutil.Encode(blockL.Signature) {
			return fmt.Errorf("prysm response %v does not match lighthouse response %v",
				blockP,
				blockL)
		}
	} else {
		blockP := &ethpb.SignedBeaconBlockBellatrix{}
		blockL := &ethpb.SignedBeaconBlockBellatrix{}
		if err := blockL.UnmarshalSSZ(sszrspL); err != nil {
			return errors.Wrap(err, "lighthouse ssz error")
		}
		if err := blockP.UnmarshalSSZ(sszrspP); err != nil {
			return errors.Wrap(err, "prysm ssz error")
		}

		if len(blockP.Signature) == 0 || len(blockL.Signature) == 0 || hexutil.Encode(blockP.Signature) != hexutil.Encode(blockL.Signature) {
			return fmt.Errorf("prysm response %v does not match lighthouse response %v",
				blockP,
				blockL)
		}
	}
	return nil
}

func compareJSONMulticlient(beaconNodeIdx int, base string, path string, respJSONPrysm interface{}, respJSONLighthouse interface{}) error {
	if err := doMiddlewareJSONGetRequest(
		base,
		path,
		beaconNodeIdx,
		respJSONPrysm,
	); err != nil {
		return errors.Wrap(err, "prysm json error")
	}

	if err := doMiddlewareJSONGetRequest(
		base,
		path,
		beaconNodeIdx,
		respJSONLighthouse,
		"lighthouse",
	); err != nil {
		return errors.Wrap(err, "lighthouse json error")
	}
	if !reflect.DeepEqual(respJSONPrysm, respJSONLighthouse) {
		p, err := json.Marshal(respJSONPrysm)
		if err != nil {
			return errors.Wrap(err, "prysm json")
		}
		l, err := json.Marshal(respJSONLighthouse)
		if err != nil {
			return errors.Wrap(err, "lighthouse json")
		}
		return fmt.Errorf("prysm response %s does not match lighthouse response %s",
			string(p),
			string(l))
	}
	return nil
}

func compareSSZMulticlient(beaconNodeIdx int, base string, path string) ([]byte, []byte, error) {
	sszrspL, err := doMiddlewareSSZGetRequest(
		base,
		path,
		beaconNodeIdx,
		"lighthouse",
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "lighthouse json error")
	}

	sszrspP, err := doMiddlewareSSZGetRequest(
		base,
		path,
		beaconNodeIdx,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "prysm json error")
	}
	if !bytes.Equal(sszrspL, sszrspP) {
		return nil, nil, fmt.Errorf("prysm ssz response %s does not match lighthouse ssz response %s",
			hexutil.Encode(sszrspP),
			hexutil.Encode(sszrspL))
	}
	fmt.Printf("prysm ssz: %v", sszrspP)
	fmt.Printf("lighthouse ssz: %v", sszrspL)
	return sszrspP, sszrspL, nil
}

func pathFromParams(path string, params []string) string {
	var apiPath string
	for index, _ := range params {
		apiPath = strings.Replace(path, fmt.Sprintf("{param%d}", index+1), params[index], 1)
	}
	return apiPath
}
