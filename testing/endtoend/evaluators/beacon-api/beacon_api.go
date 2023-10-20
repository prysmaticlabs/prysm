package beaconapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/beacon"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/builder"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/node"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/validator"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/helpers"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

var (
	errSszCast             = errors.New("ssz response is not a byte array")
	errJsonCast            = errors.New("json response has wrong structure")
	errEmptyPrysmData      = errors.New("prysm data is empty")
	errEmptyLighthouseData = errors.New("lighthouse data is empty")
)

type metadata struct {
	start            primitives.Epoch
	basepath         string
	params           func(encoding string, currentEpoch primitives.Epoch) []string
	requestObject    interface{}
	prysmResps       map[string]interface{}
	lighthouseResps  map[string]interface{}
	customEvaluation func(interface{}, interface{}) error
}

var requests = map[string]metadata{
	"/beacon/genesis": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetGenesisResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetGenesisResponse{},
		},
	},
	"/beacon/states/{param1}/root": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetStateRootResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetStateRootResponse{},
		},
	},
	"/beacon/states/{param1}/fork": {
		basepath: v1PathTemplate,
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
	"/beacon/states/{param1}/finality_checkpoints": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetFinalityCheckpointsResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetFinalityCheckpointsResponse{},
		},
	},
	// we want to test comma-separated query params
	"/beacon/states/{param1}/validators?id=0,1": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetValidatorsResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetValidatorsResponse{},
		},
	},
	"/beacon/states/{param1}/validators/{param2}": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head", "0"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetValidatorResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetValidatorResponse{},
		},
	},
	"/beacon/states/{param1}/validator_balances?id=0,1": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetValidatorBalancesResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetValidatorBalancesResponse{},
		},
	},
	"/beacon/states/{param1}/committees?index=0": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetCommitteesResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetCommitteesResponse{},
		},
	},
	"/beacon/states/{param1}/sync_committees": {
		start:    helpers.AltairE2EForkEpoch,
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetSyncCommitteeResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetSyncCommitteeResponse{},
		},
	},
	"/beacon/states/{param1}/randao": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetRandaoResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetRandaoResponse{},
		},
	},
	"/beacon/headers": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetBlockHeadersResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetBlockHeadersResponse{},
		},
	},
	"/beacon/headers/{param1}": {
		basepath: v1PathTemplate,
		params: func(_ string, e primitives.Epoch) []string {
			slot := uint64(0)
			if e > 0 {
				slot = (uint64(e) * uint64(params.BeaconConfig().SlotsPerEpoch)) - 1
			}
			return []string{fmt.Sprintf("%v", slot)}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetBlockHeaderResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetBlockHeaderResponse{},
		},
	},
	"/beacon/blocks/{param1}": {
		basepath: v2PathTemplate,
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
			"json": &beacon.GetBlockV2Response{},
			"ssz":  []byte{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetBlockV2Response{},
			"ssz":  []byte{},
		},
	},
	"/beacon/blocks/{param1}/root": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.BlockRootResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.BlockRootResponse{},
		},
	},
	"/beacon/blocks/{param1}/attestations": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &beacon.GetBlockAttestationsResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &beacon.GetBlockAttestationsResponse{},
		},
	},
	"/builder/states/{param1}/expected_withdrawals": {
		start:    helpers.CapellaE2EForkEpoch,
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &builder.ExpectedWithdrawalsResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &builder.ExpectedWithdrawalsResponse{},
		},
	},
	"/config/fork_schedule": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
		},
		prysmResps: map[string]interface{}{
			"json": &apimiddleware.ForkScheduleResponseJson{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &apimiddleware.ForkScheduleResponseJson{},
		},
		customEvaluation: func(p interface{}, l interface{}) error {
			// remove all forks with far-future epoch
			pSchedule, ok := p.(*apimiddleware.ForkScheduleResponseJson)
			if !ok {
				return errJsonCast
			}
			for i := len(pSchedule.Data) - 1; i >= 0; i-- {
				if pSchedule.Data[i].Epoch == fmt.Sprintf("%d", params.BeaconConfig().FarFutureEpoch) {
					pSchedule.Data = append(pSchedule.Data[:i], pSchedule.Data[i+1:]...)
				}
			}
			lSchedule, ok := l.(*apimiddleware.ForkScheduleResponseJson)
			if !ok {
				return errJsonCast
			}
			for i := len(lSchedule.Data) - 1; i >= 0; i-- {
				if lSchedule.Data[i].Epoch == fmt.Sprintf("%d", params.BeaconConfig().FarFutureEpoch) {
					lSchedule.Data = append(lSchedule.Data[:i], lSchedule.Data[i+1:]...)
				}
			}
			return compareJSONResponseObjects(pSchedule, lSchedule)
		},
	},
	/*"/config/spec": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
		},
		prysmResps: map[string]interface{}{
			"json": &apimiddleware.SpecResponseJson{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &apimiddleware.SpecResponseJson{},
		},
	},*/
	"/config/deposit_contract": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
		},
		prysmResps: map[string]interface{}{
			"json": &apimiddleware.DepositContractResponseJson{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &apimiddleware.DepositContractResponseJson{},
		},
	},
	"/debug/beacon/states/{param1}": {
		basepath: v2PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{"head"}
		},
		prysmResps: map[string]interface{}{
			"json": &apimiddleware.BeaconStateV2ResponseJson{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &apimiddleware.BeaconStateV2ResponseJson{},
		},
	},
	"/debug/beacon/heads": {
		basepath: v2PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
		},
		prysmResps: map[string]interface{}{
			"json": &apimiddleware.V2ForkChoiceHeadsResponseJson{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &apimiddleware.V2ForkChoiceHeadsResponseJson{},
		},
	},
	"/node/identity": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
		},
		prysmResps: map[string]interface{}{
			"json": &node.GetIdentityResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &node.GetIdentityResponse{},
		},
		customEvaluation: func(p interface{}, l interface{}) error {
			pIdentity, ok := p.(*node.GetIdentityResponse)
			if !ok {
				return errJsonCast
			}
			lIdentity, ok := l.(*node.GetIdentityResponse)
			if !ok {
				return errJsonCast
			}
			if pIdentity.Data == nil {
				return errEmptyPrysmData
			}
			if lIdentity.Data == nil {
				return errEmptyLighthouseData
			}
			return nil
		},
	},
	"/node/peers": {
		basepath: v1PathTemplate,
		params: func(_ string, _ primitives.Epoch) []string {
			return []string{}
		},
		prysmResps: map[string]interface{}{
			"json": &node.GetPeersResponse{},
		},
		lighthouseResps: map[string]interface{}{
			"json": &node.GetPeersResponse{},
		},
		customEvaluation: func(p interface{}, l interface{}) error {
			pPeers, ok := p.(*node.GetPeersResponse)
			if !ok {
				return errJsonCast
			}
			lPeers, ok := l.(*node.GetPeersResponse)
			if !ok {
				return errJsonCast
			}
			if pPeers.Data == nil {
				return errEmptyPrysmData
			}
			if lPeers.Data == nil {
				return errEmptyLighthouseData
			}
			return nil
		},
	},
	"/validator/duties/proposer/{param1}": {
		basepath: v1PathTemplate,
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
		basepath: v1PathTemplate,
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
}

func withCompareBeaconAPIs(beaconNodeIdx int) error {
	genesisResp := &beacon.GetGenesisResponse{}
	err := doJSONGetRequest(
		v1PathTemplate,
		"/beacon/genesis",
		beaconNodeIdx,
		genesisResp,
	)
	if err != nil {
		return errors.Wrap(err, "error getting genesis data")
	}
	genesisTime, err := strconv.ParseInt(genesisResp.Data.GenesisTime, 10, 64)
	if err != nil {
		return errors.Wrap(err, "could not parse genesis time")
	}
	currentEpoch := slots.EpochsSinceGenesis(time.Unix(genesisTime, 0))

	for path, meta := range requests {
		if currentEpoch < meta.start {
			continue
		}
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
					requests[path].prysmResps[key],
					requests[path].lighthouseResps[key],
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
				requests[path].prysmResps[key] = prysmr
				requests[path].lighthouseResps[key] = lighthouser
			default:
				return fmt.Errorf("unknown encoding type %s", key)
			}
		}
	}
	return postEvaluation(beaconNodeIdx, requests)
}

// postEvaluation performs additional evaluation after all requests have been completed.
// It is useful for things such as checking if specific fields match between endpoints.
func postEvaluation(beaconNodeIdx int, requests map[string]metadata) error {
	// verify that SSZ responses have the correct structure
	forkData := requests["/beacon/states/{param1}/fork"]
	fork, ok := forkData.prysmResps["json"].(*beacon.GetStateForkResponse)
	if !ok {
		return errJsonCast
	}
	finalizedEpoch, err := strconv.ParseUint(fork.Data.Epoch, 10, 64)
	if err != nil {
		return err
	}
	blockData := requests["/beacon/blocks/{param1}"]
	ssz, ok := blockData.prysmResps["ssz"].([]byte)
	if !ok {
		return errSszCast
	}
	if finalizedEpoch < helpers.AltairE2EForkEpoch+2 {
		b := &ethpb.SignedBeaconBlock{}
		if err := b.UnmarshalSSZ(ssz); err != nil {
			return errors.Wrap(err, "failed to unmarshal prysm ssz")
		}
	} else if finalizedEpoch >= helpers.AltairE2EForkEpoch+2 && finalizedEpoch < helpers.BellatrixE2EForkEpoch {
		b := &ethpb.SignedBeaconBlockAltair{}
		if err := b.UnmarshalSSZ(ssz); err != nil {
			return errors.Wrap(err, "failed to unmarshal prysm ssz")
		}
	} else if finalizedEpoch >= helpers.BellatrixE2EForkEpoch && finalizedEpoch < helpers.CapellaE2EForkEpoch {
		b := &ethpb.SignedBeaconBlockBellatrix{}
		if err := b.UnmarshalSSZ(ssz); err != nil {
			return errors.Wrap(err, "failed to unmarshal prysm ssz")
		}
	} else {
		b := &ethpb.SignedBeaconBlockCapella{}
		if err := b.UnmarshalSSZ(ssz); err != nil {
			return errors.Wrap(err, "failed to unmarshal prysm ssz")
		}
	}

	// verify that dependent root of proposer duties matches block header
	blockHeaderData := requests["/beacon/headers/{param1}"]
	header, ok := blockHeaderData.prysmResps["json"].(*beacon.GetBlockHeaderResponse)
	if !ok {
		return errJsonCast
	}
	dutiesData := requests["/validator/duties/proposer/{param1}"]
	duties, ok := dutiesData.prysmResps["json"].(*validator.GetProposerDutiesResponse)
	if !ok {
		return errJsonCast
	}
	if header.Data.Root != duties.DependentRoot {
		return fmt.Errorf("header root %s does not match duties root %s ", header.Data.Root, duties.DependentRoot)
	}

	// get first peer returned from /node/peers and use it's ID to test the /node/peers/{peer_id} endpoint
	peersData := requests["/node/peers"]
	pPeers, ok := peersData.prysmResps["json"].(*node.GetPeersResponse)
	if !ok {
		return errJsonCast
	}
	pPeer := &node.GetPeerResponse{}
	if err = doJSONGetRequest(v1PathTemplate, "/node/peers/"+pPeers.Data[0].PeerId, beaconNodeIdx, pPeer); err != nil {
		return err
	}
	if pPeer.Data == nil {
		return errEmptyPrysmData
	}
	lPeers, ok := peersData.lighthouseResps["json"].(*node.GetPeersResponse)
	if !ok {
		return errJsonCast
	}
	lPeer := &node.GetPeerResponse{}
	if err = doJSONGetRequest(v1PathTemplate, "/node/peers/"+lPeers.Data[0].PeerId, beaconNodeIdx, lPeer, "lighthouse"); err != nil {
		return err
	}
	if lPeer.Data == nil {
		return errEmptyLighthouseData
	}

	return nil
}

func compareJSONMulticlient(
	beaconNodeIdx int,
	base string,
	path string,
	requestObj, respJSONPrysm, respJSONLighthouse interface{},
	customEvaluator func(interface{}, interface{}) error,
) error {
	if requestObj != nil {
		if err := doJSONPostRequest(
			base,
			path,
			beaconNodeIdx,
			requestObj,
			respJSONPrysm,
		); err != nil {
			return errors.Wrapf(err, "could not perform Prysm JSON POST request for path %s", path)
		}

		if err := doJSONPostRequest(
			base,
			path,
			beaconNodeIdx,
			requestObj,
			respJSONLighthouse,
			"lighthouse",
		); err != nil {
			return errors.Wrapf(err, "could not perform Lighthouse JSON POST request for path %s", path)
		}
	} else {
		if err := doJSONGetRequest(
			base,
			path,
			beaconNodeIdx,
			respJSONPrysm,
		); err != nil {
			return errors.Wrapf(err, "could not perform Prysm JSON GET request for path %s", path)
		}

		if err := doJSONGetRequest(
			base,
			path,
			beaconNodeIdx,
			respJSONLighthouse,
			"lighthouse",
		); err != nil {
			return errors.Wrapf(err, "could not perform Lighthouse JSON GET request for path %s", path)
		}
	}
	if customEvaluator != nil {
		return customEvaluator(respJSONPrysm, respJSONLighthouse)
	} else {
		return compareJSONResponseObjects(respJSONPrysm, respJSONLighthouse)
	}
}

func compareSSZMulticlient(beaconNodeIdx int, base string, path string) ([]byte, []byte, error) {
	sszrspL, err := doSSZGetRequest(
		base,
		path,
		beaconNodeIdx,
		"lighthouse",
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not perform GET request for Lighthouse SSZ")
	}

	sszrspP, err := doSSZGetRequest(
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
		apiPath = strings.Replace(apiPath, fmt.Sprintf("{param%d}", index+1), params[index], 1)
	}
	return apiPath
}
