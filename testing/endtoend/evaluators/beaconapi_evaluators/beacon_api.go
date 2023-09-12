package beaconapi_evaluators

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/beacon"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/validator"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/proto/eth/service"
	v1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/helpers"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"google.golang.org/grpc"
)

type metadata struct {
	basepath         string
	params           func(encoding string, currentEpoch primitives.Epoch) []string
	requestObject    interface{}
	prysmResps       map[string]interface{}
	lighthouseResps  map[string]interface{}
	customEvaluation func(interface{}, interface{}) error
}

var beaconPathsAndObjects = map[string]metadata{
	"/beacon/genesis": {
		basepath: v1MiddlewarePathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
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
		params: func(_ string, _ primitives.Epoch) []string {
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
		params: func(_ string, _ primitives.Epoch) []string {
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
		params: func(t string, e primitives.Epoch) []string {
			if t == "ssz" {
				if e < 4 {
					return []string{"genesis"}
				}
				return []string{"finalized"}
			}
			return []string{"head"}

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
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"finalized"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetStateForkResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetStateForkResponse{},
		},
	},
	"/debug/beacon/states/{param1}": {
		basepath: v2MiddlewarePathTemplate,
		params: func(_ string, e primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &apimiddleware.BeaconStateV2ResponseJson{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &apimiddleware.BeaconStateV2ResponseJson{},
		},
	},
	"/validator/duties/proposer/{param1}": {
		basepath: v1MiddlewarePathTemplate,
		params: func(_ string, e primitives.Epoch) []string {
			return []string{fmt.Sprintf("%v", e)}
		},
		prysmResps: map[string]interface{}{
			"json": &validator.GetProposerDutiesResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &validator.GetProposerDutiesResponse{},
		},
		customEvaluation: func(prysmResp interface{}, lhouseResp interface{}) error {
			castedl, ok := lhouseResp.(*validator.GetProposerDutiesResponse)
			if !ok {
				return errors.New("failed to cast type")
			}
			if castedl.Data[0].Slot == "0" {
				// remove the first item from lighthouse data since lighthouse is returning a value despite no proposer
				// there is no proposer on slot 0 so prysm don't return anything for slot 0
				castedl.Data = castedl.Data[1:]
			}
			return compareJSONResponseObjects(prysmResp, castedl)
		},
	},
	"/validator/duties/attester/{param1}": {
		basepath: v1MiddlewarePathTemplate,
		params: func(_ string, e primitives.Epoch) []string {
			//ask for a future epoch to test this case
			return []string{fmt.Sprintf("%v", e+1)}
		},
		requestObject: func() []string {
			validatorIndices := make([]string, 64)
			for key := range validatorIndices {
				validatorIndices[key] = fmt.Sprintf("%d", key)
			}
			return validatorIndices
		}(),
		prysmResps: map[string]interface{}{
			"json": &validator.GetAttesterDutiesResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &validator.GetAttesterDutiesResponse{},
		},
		customEvaluation: func(prysmResp interface{}, lhouseResp interface{}) error {
			castedp, ok := lhouseResp.(*validator.GetAttesterDutiesResponse)
			if !ok {
				return errors.New("failed to cast type")
			}
			castedl, ok := lhouseResp.(*validator.GetAttesterDutiesResponse)
			if !ok {
				return errors.New("failed to cast type")
			}
			if len(castedp.Data) == 0 ||
				len(castedl.Data) == 0 ||
				len(castedp.Data) != len(castedl.Data) {
				return fmt.Errorf("attester data does not match, prysm: %d lighthouse: %d", len(castedp.Data), len(castedl.Data))
			}
			return compareJSONResponseObjects(prysmResp, castedl)
		},
	},
	"/beacon/headers/{param1}": {
		basepath: v1MiddlewarePathTemplate,
		params: func(_ string, e primitives.Epoch) []string {
			slot := uint64(0)
			if e > 0 {
				slot = (uint64(e) * uint64(params.BeaconConfig().SlotsPerEpoch)) - 1
			}
			return []string{fmt.Sprintf("%v", slot)}
		},
		prysmResps: map[string]interface{}{
			"json": &apimiddleware.BlockHeaderResponseJson{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &apimiddleware.BlockHeaderResponseJson{},
		},
	},
	"/node/identity": {
		basepath: v1MiddlewarePathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
		},
		prysmResps: map[string]interface{}{
			"json": &apimiddleware.IdentityResponseJson{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &apimiddleware.IdentityResponseJson{},
		},
		customEvaluation: func(prysmResp interface{}, lhouseResp interface{}) error {
			castedp, ok := prysmResp.(*apimiddleware.IdentityResponseJson)
			if !ok {
				return errors.New("failed to cast type")
			}
			castedl, ok := lhouseResp.(*apimiddleware.IdentityResponseJson)
			if !ok {
				return errors.New("failed to cast type")
			}
			if castedp.Data == nil {
				return errors.New("prysm node identity was empty")
			}
			if castedl.Data == nil {
				return errors.New("lighthouse node identity was empty")
			}
			return nil
		},
	},
	"/node/peers": {
		basepath: v1MiddlewarePathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
		},
		prysmResps: map[string]interface{}{
			"json": &apimiddleware.PeersResponseJson{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &apimiddleware.PeersResponseJson{},
		},
		customEvaluation: func(prysmResp interface{}, lhouseResp interface{}) error {
			castedp, ok := prysmResp.(*apimiddleware.PeersResponseJson)
			if !ok {
				return errors.New("failed to cast type")
			}
			castedl, ok := lhouseResp.(*apimiddleware.PeersResponseJson)
			if !ok {
				return errors.New("failed to cast type")
			}
			if castedp.Data == nil {
				return errors.New("prysm node identity was empty")
			}
			if castedl.Data == nil {
				return errors.New("lighthouse node identity was empty")
			}
			return nil
		},
	},
}

func withCompareBeaconAPIs(beaconNodeIdx int, conn *grpc.ClientConn) error {
	ctx := context.Background()
	beaconClient := service.NewBeaconChainClient(conn)
	genesisData, err := beaconClient.GetGenesis(ctx, &empty.Empty{})
	if err != nil {
		return errors.Wrap(err, "error getting genesis data")
	}
	currentEpoch := slots.EpochsSinceGenesis(genesisData.Data.GenesisTime.AsTime())

	for path, meta := range beaconPathsAndObjects {
		for key := range meta.prysmResps {
			switch key {
			case "json":
				jsonparams := meta.params("json", currentEpoch)
				apipath := pathFromParams(path, jsonparams)
				fmt.Printf("executing json api path: %s\n", apipath)
				if err := compareJSONMulticlient(beaconNodeIdx,
					meta.basepath,
					apipath,
					meta.requestObject,
					beaconPathsAndObjects[path].prysmResps[key],
					beaconPathsAndObjects[path].lighthouseResps[key],
					meta.customEvaluation,
				); err != nil {
					return err
				}
			case "ssz":
				sszparams := meta.params("ssz", currentEpoch)
				if len(sszparams) == 0 {
					continue
				}
				apipath := pathFromParams(path, sszparams)
				fmt.Printf("executing ssz api path: %s\n", apipath)
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
	return orderedEvaluationOnResponses(beaconPathsAndObjects, genesisData)
}

func orderedEvaluationOnResponses(beaconPathsAndObjects map[string]metadata, genesisData *v1.GenesisResponse) error {
	forkPathData := beaconPathsAndObjects["/beacon/states/{param1}/fork"]
	prysmForkData, ok := forkPathData.prysmResps["json"].(*beacon.GetStateForkResponse)
	if !ok {
		return errors.New("failed to cast type")
	}
	lighthouseForkData, ok := forkPathData.lighthouseResps["json"].(*beacon.GetStateForkResponse)
	if !ok {
		return errors.New("failed to cast type")
	}
	if prysmForkData.Data.Epoch != lighthouseForkData.Data.Epoch {
		return fmt.Errorf("prysm epoch %v does not match lighthouse epoch %v",
			prysmForkData.Data.Epoch,
			lighthouseForkData.Data.Epoch)
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
	if finalizedEpoch < helpers.AltairE2EForkEpoch+2 {
		blockP := &ethpb.SignedBeaconBlock{}
		blockL := &ethpb.SignedBeaconBlock{}
		if err := blockL.UnmarshalSSZ(sszrspL); err != nil {
			return errors.Wrap(err, "failed to unmarshal lighthouse ssz")
		}
		if err := blockP.UnmarshalSSZ(sszrspP); err != nil {
			return errors.Wrap(err, "failed to unmarshal rysm ssz")
		}
		if len(blockP.Signature) == 0 || len(blockL.Signature) == 0 || hexutil.Encode(blockP.Signature) != hexutil.Encode(blockL.Signature) {
			return errors.New("prysm signature does not match lighthouse signature")
		}
	} else if finalizedEpoch >= helpers.AltairE2EForkEpoch+2 && finalizedEpoch < helpers.BellatrixE2EForkEpoch {
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
	blockheaderData := beaconPathsAndObjects["/beacon/headers/{param1}"]
	prysmHeader, ok := blockheaderData.prysmResps["json"].(*apimiddleware.BlockHeaderResponseJson)
	if !ok {
		return errors.New("failed to cast type")
	}
	proposerdutiesData := beaconPathsAndObjects["/validator/duties/proposer/{param1}"]
	prysmDuties, ok := proposerdutiesData.prysmResps["json"].(*validator.GetProposerDutiesResponse)
	if !ok {
		return errors.New("failed to cast type")
	}
	if prysmHeader.Data.Root != prysmDuties.DependentRoot {
		fmt.Printf("current slot: %v\n", slots.CurrentSlot(uint64(genesisData.Data.GenesisTime.AsTime().Unix())))
		return fmt.Errorf("header root %s does not match duties root %s ", prysmHeader.Data.Root, prysmDuties.DependentRoot)
	}

	return nil
}

func compareJSONMulticlient(beaconNodeIdx int, base string, path string, requestObj, respJSONPrysm interface{}, respJSONLighthouse interface{}, customEvaluator func(interface{}, interface{}) error) error {
	if requestObj != nil {
		if err := doMiddlewareJSONPostRequest(
			base,
			path,
			beaconNodeIdx,
			requestObj,
			respJSONPrysm,
		); err != nil {
			return errors.Wrap(err, "could not perform POST request for Prysm JSON")
		}

		if err := doMiddlewareJSONPostRequest(
			base,
			path,
			beaconNodeIdx,
			requestObj,
			respJSONLighthouse,
			"lighthouse",
		); err != nil {
			return errors.Wrap(err, "could not perform POST request for Lighthouse JSON")
		}
	} else {
		if err := doMiddlewareJSONGetRequest(
			base,
			path,
			beaconNodeIdx,
			respJSONPrysm,
		); err != nil {
			return errors.Wrap(err, "could not perform GET request for Prysm JSON")
		}

		if err := doMiddlewareJSONGetRequest(
			base,
			path,
			beaconNodeIdx,
			respJSONLighthouse,
			"lighthouse",
		); err != nil {
			return errors.Wrap(err, "could not perform GET request for Lighthouse JSON")
		}
	}
	if customEvaluator != nil {
		return customEvaluator(respJSONPrysm, respJSONLighthouse)
	} else {
		return compareJSONResponseObjects(respJSONPrysm, respJSONLighthouse)
	}
}

func compareSSZMulticlient(beaconNodeIdx int, base string, path string) ([]byte, []byte, error) {
	sszrspL, err := doMiddlewareSSZGetRequest(
		base,
		path,
		beaconNodeIdx,
		"lighthouse",
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not perform GET request for Lighthouse SSZ")
	}

	sszrspP, err := doMiddlewareSSZGetRequest(
		base,
		path,
		beaconNodeIdx,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not perform GET request for Prysm SSZ")
	}
	if !bytes.Equal(sszrspL, sszrspP) {
		return nil, nil, errors.New("prysm ssz response does not match lighthouse ssz response")
	}
	return sszrspP, sszrspL, nil
}

func compareJSONResponseObjects(prysmResp interface{}, lighthouseResp interface{}) error {
	if !reflect.DeepEqual(prysmResp, lighthouseResp) {
		p, err := json.Marshal(prysmResp)
		if err != nil {
			return errors.Wrap(err, "failed to marshal Prysm response to JSON")
		}
		l, err := json.Marshal(lighthouseResp)
		if err != nil {
			return errors.Wrap(err, "failed to marshal Lighthouse response to JSON")
		}
		return fmt.Errorf("prysm response %s does not match lighthouse response %s",
			string(p),
			string(l))
	}
	return nil
}

func pathFromParams(path string, params []string) string {
	apiPath := path
	for index := range params {
		apiPath = strings.Replace(path, fmt.Sprintf("{param%d}", index+1), params[index], 1)
	}
	return apiPath
}
