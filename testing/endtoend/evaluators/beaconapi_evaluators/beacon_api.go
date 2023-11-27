package beaconapi_evaluators

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/beacon"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/config"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/debug"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/node"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/validator"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/helpers"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

var (
	errSszCast             = errors.New("ssz response is not a byte array")
	errEmptyPrysmData      = errors.New("prysm data is empty")
	errEmptyLighthouseData = errors.New("lighthouse data is empty")
)

const (
	msgWrongJson = "json response has wrong structure, expected %T, got %T"
)

type meta interface {
	getStart() primitives.Epoch
	setStart(start primitives.Epoch)
	getBasePath() string
	getReq() interface{}
	setReq(req interface{})
	getPResp() interface{}
	getLResp() interface{}
	getParams(epoch primitives.Epoch) []string
	setParams(f func(primitives.Epoch) []string)
	getCustomEval() func(interface{}, interface{}) error
	setCustomEval(f func(interface{}, interface{}) error)
}

type metadata[Resp any] struct {
	start      primitives.Epoch
	basePath   string
	req        interface{}
	pResp      *Resp
	lResp      *Resp
	params     func(currentEpoch primitives.Epoch) []string
	customEval func(interface{}, interface{}) error
}

func (m *metadata[Resp]) getStart() primitives.Epoch {
	return m.start
}

func (m *metadata[Resp]) setStart(start primitives.Epoch) {
	m.start = start
}

func (m *metadata[Resp]) getBasePath() string {
	return m.basePath
}

func (m *metadata[Resp]) getReq() interface{} {
	return m.req
}

func (m *metadata[Resp]) setReq(req interface{}) {
	m.req = req
}

func (m *metadata[Resp]) getPResp() interface{} {
	return m.pResp
}

func (m *metadata[Resp]) getLResp() interface{} {
	return m.lResp
}

func (m *metadata[Resp]) getParams(epoch primitives.Epoch) []string {
	if m.params == nil {
		return nil
	}
	return m.params(epoch)
}

func (m *metadata[Resp]) setParams(f func(currentEpoch primitives.Epoch) []string) {
	m.params = f
}

func (m *metadata[Resp]) getCustomEval() func(interface{}, interface{}) error {
	return m.customEval
}

func (m *metadata[Resp]) setCustomEval(f func(interface{}, interface{}) error) {
	m.customEval = f
}

func newMetadata[Resp any](basePath string, opts ...metadataOpt) *metadata[Resp] {
	m := &metadata[Resp]{
		basePath: basePath,
		pResp:    new(Resp),
		lResp:    new(Resp),
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

type metadataOpt func(meta)

func withStart(start primitives.Epoch) metadataOpt {
	return func(m meta) {
		m.setStart(start)
	}
}

func withReq(req interface{}) metadataOpt {
	return func(m meta) {
		m.setReq(req)
	}
}

func withParams(f func(currentEpoch primitives.Epoch) []string) metadataOpt {
	return func(m meta) {
		m.setParams(f)
	}
}

func withCustomEval(f func(interface{}, interface{}) error) metadataOpt {
	return func(m meta) {
		m.setCustomEval(f)
	}
}

var requests = map[string]meta{
	"/beacon/genesis": newMetadata[beacon.GetGenesisResponse](v1PathTemplate),
	"/beacon/states/{param1}/root": newMetadata[beacon.GetStateRootResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/fork": newMetadata[beacon.GetStateForkResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"finalized"}
		})),
	"/beacon/states/{param1}/finality_checkpoints": newMetadata[beacon.GetFinalityCheckpointsResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	// we want to test comma-separated query params
	"/beacon/states/{param1}/validators?id=0,1": newMetadata[beacon.GetValidatorsResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/validators/{param2}": newMetadata[beacon.GetValidatorResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head", "0"}
		})),
	"/beacon/states/{param1}/validator_balances?id=0,1": newMetadata[beacon.GetValidatorBalancesResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/committees?index=0": newMetadata[beacon.GetCommitteesResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/sync_committees": newMetadata[beacon.GetSyncCommitteeResponse](v1PathTemplate,
		withStart(helpers.AltairE2EForkEpoch),
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/randao": newMetadata[beacon.GetRandaoResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/headers": newMetadata[beacon.GetBlockHeadersResponse](v1PathTemplate),
	"/beacon/headers/{param1}": newMetadata[beacon.GetBlockHeaderResponse](v1PathTemplate,
		withParams(func(e primitives.Epoch) []string {
			slot := uint64(0)
			if e > 0 {
				slot = (uint64(e) * uint64(params.BeaconConfig().SlotsPerEpoch)) - 1
			}
			return []string{fmt.Sprintf("%v", slot)}
		})),
	"/beacon/blocks/{param1}": newMetadata[beacon.GetBlockV2Response](v2PathTemplate,
		withParams(func(e primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/blocks/{param1}/root": newMetadata[beacon.BlockRootResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/blocks/{param1}/attestations": newMetadata[beacon.GetBlockAttestationsResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/blinded_blocks/{param1}": newMetadata[beacon.GetBlockV2Response](v1PathTemplate,
		withParams(func(e primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/pool/attestations": newMetadata[beacon.ListAttestationsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*beacon.ListAttestationsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.ListAttestationsResponse{}, p)
			}
			lResp, ok := l.(*beacon.ListAttestationsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.ListAttestationsResponse{}, l)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lResp.Data == nil {
				return errEmptyLighthouseData
			}
			return nil
		})),
	"/beacon/pool/attester_slashings": newMetadata[beacon.GetAttesterSlashingsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*beacon.GetAttesterSlashingsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.GetAttesterSlashingsResponse{}, p)
			}
			lResp, ok := l.(*beacon.GetAttesterSlashingsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.GetAttesterSlashingsResponse{}, l)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lResp.Data == nil {
				return errEmptyLighthouseData
			}
			return nil
		})),
	"/beacon/pool/proposer_slashings": newMetadata[beacon.GetProposerSlashingsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*beacon.GetProposerSlashingsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.GetProposerSlashingsResponse{}, p)
			}
			lResp, ok := l.(*beacon.GetProposerSlashingsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.GetProposerSlashingsResponse{}, l)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lResp.Data == nil {
				return errEmptyLighthouseData
			}
			return nil
		})),
	"/beacon/pool/voluntary_exits": newMetadata[beacon.ListVoluntaryExitsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*beacon.ListVoluntaryExitsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.ListVoluntaryExitsResponse{}, p)
			}
			lResp, ok := l.(*beacon.ListVoluntaryExitsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.ListVoluntaryExitsResponse{}, l)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lResp.Data == nil {
				return errEmptyLighthouseData
			}
			return nil
		})),
	"/beacon/pool/bls_to_execution_changes": newMetadata[beacon.BLSToExecutionChangesPoolResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*beacon.BLSToExecutionChangesPoolResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.BLSToExecutionChangesPoolResponse{}, p)
			}
			lResp, ok := l.(*beacon.BLSToExecutionChangesPoolResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.BLSToExecutionChangesPoolResponse{}, l)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lResp.Data == nil {
				return errEmptyLighthouseData
			}
			return nil
		})),
	"/config/fork_schedule": newMetadata[config.GetForkScheduleResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*config.GetForkScheduleResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &config.GetForkScheduleResponse{}, p)
			}
			lResp, ok := l.(*config.GetForkScheduleResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &config.GetForkScheduleResponse{}, l)
			}
			// remove all forks with far-future epoch
			for i := len(pResp.Data) - 1; i >= 0; i-- {
				if pResp.Data[i].Epoch == fmt.Sprintf("%d", params.BeaconConfig().FarFutureEpoch) {
					pResp.Data = append(pResp.Data[:i], pResp.Data[i+1:]...)
				}
			}
			for i := len(lResp.Data) - 1; i >= 0; i-- {
				if lResp.Data[i].Epoch == fmt.Sprintf("%d", params.BeaconConfig().FarFutureEpoch) {
					lResp.Data = append(lResp.Data[:i], lResp.Data[i+1:]...)
				}
			}
			return compareJSON(pResp, lResp)
		})),
	"/config/deposit_contract": newMetadata[config.GetDepositContractResponse](v1PathTemplate),
	"/debug/beacon/states/{param1}": newMetadata[debug.GetBeaconStateV2Response](v2PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/debug/beacon/heads": newMetadata[debug.GetForkChoiceHeadsV2Response](v2PathTemplate),
	"/node/identity": newMetadata[node.GetIdentityResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*node.GetIdentityResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &node.GetIdentityResponse{}, p)
			}
			lResp, ok := l.(*node.GetIdentityResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &node.GetIdentityResponse{}, l)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lResp.Data == nil {
				return errEmptyLighthouseData
			}
			return nil
		})),
	"/node/peers": newMetadata[node.GetPeersResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*node.GetPeersResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &node.GetPeersResponse{}, p)
			}
			lResp, ok := l.(*node.GetPeersResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &node.GetPeersResponse{}, l)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lResp.Data == nil {
				return errEmptyLighthouseData
			}
			return nil
		})),
	"/node/peer_count": newMetadata[node.GetPeerCountResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*node.GetPeerCountResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &node.GetPeerCountResponse{}, p)
			}
			lResp, ok := l.(*node.GetPeerCountResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &node.GetPeerCountResponse{}, l)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lResp.Data == nil {
				return errEmptyLighthouseData
			}
			return nil
		})),
	"/node/version": newMetadata[node.GetVersionResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*node.GetVersionResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.ListAttestationsResponse{}, p)
			}
			lResp, ok := l.(*node.GetVersionResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.ListAttestationsResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if !strings.Contains(pResp.Data.Version, "Prysm") {
				return errors.New("version response does not contain Prysm client name")
			}
			if lResp.Data == nil {
				return errEmptyLighthouseData
			}
			if !strings.Contains(lResp.Data.Version, "Lighthouse") {
				return errors.New("version response does not contain Lighthouse client name")
			}
			return nil
		})),
	"/node/syncing": newMetadata[node.SyncStatusResponse](v1PathTemplate),
	"/validator/duties/proposer/{param1}": newMetadata[validator.GetProposerDutiesResponse](v1PathTemplate,
		withParams(func(e primitives.Epoch) []string {
			return []string{fmt.Sprintf("%v", e)}
		}),
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*validator.GetProposerDutiesResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &validator.GetProposerDutiesResponse{}, p)
			}
			lResp, ok := l.(*validator.GetProposerDutiesResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &validator.GetProposerDutiesResponse{}, l)
			}
			if lResp.Data[0].Slot == "0" {
				// remove the first item from lighthouse data since lighthouse is returning a value despite no proposer
				// there is no proposer on slot 0 so prysm don't return anything for slot 0
				lResp.Data = lResp.Data[1:]
			}
			return compareJSON(pResp, lResp)
		})),
	"/validator/duties/attester/{param1}": newMetadata[validator.GetAttesterDutiesResponse](v1PathTemplate,
		withParams(func(e primitives.Epoch) []string {
			//ask for a future epoch to test this case
			return []string{fmt.Sprintf("%v", e+1)}
		}),
		withReq(func() []string {
			validatorIndices := make([]string, 64)
			for key := range validatorIndices {
				validatorIndices[key] = fmt.Sprintf("%d", key)
			}
			return validatorIndices
		}()),
		withCustomEval(func(p interface{}, l interface{}) error {
			pResp, ok := p.(*validator.GetAttesterDutiesResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &validator.GetAttesterDutiesResponse{}, p)
			}
			lResp, ok := l.(*validator.GetAttesterDutiesResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &validator.GetAttesterDutiesResponse{}, l)
			}
			if len(pResp.Data) == 0 ||
				len(lResp.Data) == 0 ||
				len(pResp.Data) != len(lResp.Data) {
				return fmt.Errorf("attester data does not match, prysm: %d lighthouse: %d", len(pResp.Data), len(lResp.Data))
			}
			return compareJSON(pResp, lResp)
		})),
}

func withCompareBeaconAPIs(nodeIdx int) error {
	genesisResp := &beacon.GetGenesisResponse{}
	if err := doJSONGetRequest(v1PathTemplate, "/beacon/genesis", nodeIdx, genesisResp); err != nil {
		return errors.Wrap(err, "error getting genesis data")
	}
	genesisTime, err := strconv.ParseInt(genesisResp.Data.GenesisTime, 10, 64)
	if err != nil {
		return errors.Wrap(err, "could not parse genesis time")
	}
	currentEpoch := slots.EpochsSinceGenesis(time.Unix(genesisTime, 0))

	for path, m := range requests {
		if currentEpoch < m.getStart() {
			continue
		}
		apiPath := path
		if m.getParams(currentEpoch) != nil {
			apiPath = pathFromParams(path, m.getParams(currentEpoch))
		}
		if err = compareJSONMultiClient(nodeIdx, m.getBasePath(), apiPath, m.getReq(), m.getPResp(), m.getLResp(), m.getCustomEval()); err != nil {
			return err
		}
		fmt.Printf("executing json api path: %s\n", apiPath)
	}

	return postEvaluation(requests)
}

// postEvaluation performs additional evaluation after all requests have been completed.
// It is useful for things such as checking if specific fields match between endpoints.
func postEvaluation(requests map[string]meta) error {
	// verify that block SSZ responses have the correct structure
	/*forkData := requests["/beacon/states/{param1}/fork"]
	fork, ok := forkData.getPResp().(beacon.GetStateForkResponse)
	if !ok {
		return errJsonCast
	}
	finalizedEpoch, err := strconv.ParseUint(fork.Data.Epoch, 10, 64)
	if err != nil {
		return err
	}
	blockData := requests["/beacon/blocks/{param1}"]
	blockSsz, ok := blockData.prysmResps["ssz"].([]byte)
	if !ok {
		return errSszCast
	}
	blindedBlockData := requests["/beacon/blinded_blocks/{param1}"]
	blindedBlockSsz, ok := blindedBlockData.prysmResps["ssz"].([]byte)
	if !ok {
		return errSszCast
	}
	if finalizedEpoch < helpers.AltairE2EForkEpoch+2 {
		b := &ethpb.SignedBeaconBlock{}
		if err := b.UnmarshalSSZ(blockSsz); err != nil {
			return errors.Wrap(err, "failed to unmarshal ssz")
		}
		bb := &ethpb.SignedBeaconBlock{}
		if err := bb.UnmarshalSSZ(blindedBlockSsz); err != nil {
			return errors.Wrap(err, "failed to unmarshal ssz")
		}
	} else if finalizedEpoch >= helpers.AltairE2EForkEpoch+2 && finalizedEpoch < helpers.BellatrixE2EForkEpoch {
		b := &ethpb.SignedBeaconBlockAltair{}
		if err := b.UnmarshalSSZ(blockSsz); err != nil {
			return errors.Wrap(err, "failed to unmarshal ssz")
		}
		bb := &ethpb.SignedBeaconBlockAltair{}
		if err := bb.UnmarshalSSZ(blindedBlockSsz); err != nil {
			return errors.Wrap(err, "failed to unmarshal ssz")
		}
	} else if finalizedEpoch >= helpers.BellatrixE2EForkEpoch && finalizedEpoch < helpers.CapellaE2EForkEpoch {
		b := &ethpb.SignedBeaconBlockBellatrix{}
		if err := b.UnmarshalSSZ(blockSsz); err != nil {
			return errors.Wrap(err, "failed to unmarshal ssz")
		}
		bb := &ethpb.SignedBlindedBeaconBlockBellatrix{}
		if err := bb.UnmarshalSSZ(blindedBlockSsz); err != nil {
			return errors.Wrap(err, "failed to unmarshal ssz")
		}
	} else if finalizedEpoch >= helpers.CapellaE2EForkEpoch && finalizedEpoch < helpers.DenebE2EForkEpoch {
		b := &ethpb.SignedBeaconBlockCapella{}
		if err := b.UnmarshalSSZ(blockSsz); err != nil {
			return errors.Wrap(err, "failed to unmarshal ssz")
		}
		bb := &ethpb.SignedBlindedBeaconBlockCapella{}
		if err := bb.UnmarshalSSZ(blindedBlockSsz); err != nil {
			return errors.Wrap(err, "failed to unmarshal ssz")
		}
	} else {
		b := &ethpb.SignedBeaconBlockDeneb{}
		if err := b.UnmarshalSSZ(blockSsz); err != nil {
			return errors.Wrap(err, "failed to unmarshal ssz")
		}
		bb := &ethpb.SignedBlindedBeaconBlockDeneb{}
		if err := bb.UnmarshalSSZ(blindedBlockSsz); err != nil {
			return errors.Wrap(err, "failed to unmarshal ssz")
		}
	}*/

	// verify that dependent root of proposer duties matches block header
	blockHeaderData := requests["/beacon/headers/{param1}"]
	header, ok := blockHeaderData.getPResp().(*beacon.GetBlockHeaderResponse)
	if !ok {
		return fmt.Errorf(msgWrongJson, &beacon.GetBlockHeaderResponse{}, blockHeaderData.getPResp())
	}
	dutiesData := requests["/validator/duties/proposer/{param1}"]
	duties, ok := dutiesData.getPResp().(*validator.GetProposerDutiesResponse)
	if !ok {
		return fmt.Errorf(msgWrongJson, &validator.GetProposerDutiesResponse{}, dutiesData.getPResp())
	}
	if header.Data.Root != duties.DependentRoot {
		return fmt.Errorf("header root %s does not match duties root %s ", header.Data.Root, duties.DependentRoot)
	}

	return nil
}

func compareJSONMultiClient(nodeIdx int, base, path string, req, pResp, lResp interface{}, customEval func(interface{}, interface{}) error) error {
	if req != nil {
		if err := doJSONPostRequest(base, path, nodeIdx, req, pResp); err != nil {
			return errors.Wrapf(err, "could not perform Prysm JSON POST request for path %s", path)
		}
		if err := doJSONPostRequest(base, path, nodeIdx, req, lResp, "lighthouse"); err != nil {
			return errors.Wrapf(err, "could not perform Lighthouse JSON POST request for path %s", path)
		}
	} else {
		if err := doJSONGetRequest(base, path, nodeIdx, pResp); err != nil {
			return errors.Wrapf(err, "could not perform Prysm JSON GET request for path %s", path)
		}
		if err := doJSONGetRequest(base, path, nodeIdx, lResp, "lighthouse"); err != nil {
			return errors.Wrapf(err, "could not perform Lighthouse JSON GET request for path %s", path)
		}
	}
	if customEval != nil {
		return customEval(pResp, lResp)
	} else {
		return compareJSON(pResp, lResp)
	}
}

func compareSSZMultiClient(nodeIdx int, base, path string) ([]byte, []byte, error) {
	pResp, err := doSSZGetRequest(base, path, nodeIdx)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not perform Prysm SSZ GET request for path %s", path)
	}
	lResp, err := doSSZGetRequest(base, path, nodeIdx, "lighthouse")
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not perform Lighthouse SSZ GET request for path %s", path)
	}
	if !bytes.Equal(pResp, lResp) {
		return nil, nil, errors.New("Prysm SSZ response does not match Lighthouse SSZ response")
	}
	return pResp, lResp, nil
}

func compareJSON(pResp interface{}, lResp interface{}) error {
	if !reflect.DeepEqual(pResp, lResp) {
		p, err := json.Marshal(pResp)
		if err != nil {
			return errors.Wrap(err, "failed to marshal Prysm response to JSON")
		}
		l, err := json.Marshal(lResp)
		if err != nil {
			return errors.Wrap(err, "failed to marshal Lighthouse response to JSON")
		}
		return fmt.Errorf("Prysm response %s does not match Lighthouse response %s", string(p), string(l))
	}
	return nil
}

func pathFromParams(path string, params []string) string {
	apiPath := path
	for i := range params {
		apiPath = strings.Replace(apiPath, fmt.Sprintf("{param%d}", i+1), params[i], 1)
	}
	return apiPath
}
