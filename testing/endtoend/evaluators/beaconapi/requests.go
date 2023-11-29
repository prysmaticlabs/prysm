package beaconapi

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/beacon"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/config"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/debug"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/node"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/validator"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/helpers"
)

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
		withSsz(),
		withParams(func(e primitives.Epoch) []string {
			if e < 4 {
				return []string{"head"}
			}
			return []string{"finalized"}
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
		withSsz(),
		withParams(func(e primitives.Epoch) []string {
			if e < 4 {
				return []string{"head"}
			}
			return []string{"finalized"}
		})),
	"/beacon/pool/attestations": newMetadata[beacon.ListAttestationsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*beacon.ListAttestationsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.ListAttestationsResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyData
			}
			return nil
		})),
	"/beacon/pool/attester_slashings": newMetadata[beacon.GetAttesterSlashingsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*beacon.GetAttesterSlashingsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.GetAttesterSlashingsResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyData
			}
			return nil
		})),
	"/beacon/pool/proposer_slashings": newMetadata[beacon.GetProposerSlashingsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*beacon.GetProposerSlashingsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.GetProposerSlashingsResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyData
			}
			return nil
		})),
	"/beacon/pool/voluntary_exits": newMetadata[beacon.ListVoluntaryExitsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*beacon.ListVoluntaryExitsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.ListVoluntaryExitsResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyData
			}
			return nil
		})),
	"/beacon/pool/bls_to_execution_changes": newMetadata[beacon.BLSToExecutionChangesPoolResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*beacon.BLSToExecutionChangesPoolResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &beacon.BLSToExecutionChangesPoolResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyData
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
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*node.GetIdentityResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &node.GetIdentityResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyData
			}
			return nil
		})),
	"/node/peers": newMetadata[node.GetPeersResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*node.GetPeersResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &node.GetPeersResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyData
			}
			return nil
		})),
	"/node/peer_count": newMetadata[node.GetPeerCountResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*node.GetPeerCountResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &node.GetPeerCountResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyData
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
				return errEmptyData
			}
			if !strings.Contains(pResp.Data.Version, "Prysm") {
				return errors.New("version response does not contain Prysm client name")
			}
			if lResp.Data == nil {
				return errEmptyData
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
