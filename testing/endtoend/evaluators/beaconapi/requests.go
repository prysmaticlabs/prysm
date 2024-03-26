package beaconapi

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

var requests = map[string]endpoint{
	"/beacon/genesis": newMetadata[structs.GetGenesisResponse](v1PathTemplate),
	"/beacon/states/{param1}/root": newMetadata[structs.GetStateRootResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/fork": newMetadata[structs.GetStateForkResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/finality_checkpoints": newMetadata[structs.GetFinalityCheckpointsResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	// we want to test comma-separated query params
	"/beacon/states/{param1}/validators?id=0,1": newMetadata[structs.GetValidatorsResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/validators/{param2}": newMetadata[structs.GetValidatorResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head", "0"}
		})),
	"/beacon/states/{param1}/validator_balances?id=0,1": newMetadata[structs.GetValidatorBalancesResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/committees?index=0": newMetadata[structs.GetCommitteesResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/sync_committees": newMetadata[structs.GetSyncCommitteeResponse](v1PathTemplate,
		withStart(params.BeaconConfig().AltairForkEpoch),
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/states/{param1}/randao": newMetadata[structs.GetRandaoResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/headers": newMetadata[structs.GetBlockHeadersResponse](v1PathTemplate),
	"/beacon/headers/{param1}": newMetadata[structs.GetBlockHeaderResponse](v1PathTemplate,
		withParams(func(e primitives.Epoch) []string {
			slot := uint64(0)
			if e > 0 {
				slot = (uint64(e) * uint64(params.BeaconConfig().SlotsPerEpoch)) - 1
			}
			return []string{fmt.Sprintf("%v", slot)}
		})),
	"/beacon/blocks/{param1}": newMetadata[structs.GetBlockV2Response](v2PathTemplate,
		withSsz(),
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/blocks/{param1}/root": newMetadata[structs.BlockRootResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/blocks/{param1}/attestations": newMetadata[structs.GetBlockAttestationsResponse](v1PathTemplate,
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/blinded_blocks/{param1}": newMetadata[structs.GetBlockV2Response](v1PathTemplate,
		withSsz(),
		withParams(func(_ primitives.Epoch) []string {
			return []string{"head"}
		})),
	"/beacon/pool/attestations": newMetadata[structs.ListAttestationsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*structs.ListAttestationsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.ListAttestationsResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			return nil
		})),
	"/beacon/pool/attester_slashings": newMetadata[structs.GetAttesterSlashingsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*structs.GetAttesterSlashingsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetAttesterSlashingsResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			return nil
		})),
	"/beacon/pool/proposer_slashings": newMetadata[structs.GetProposerSlashingsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*structs.GetProposerSlashingsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetProposerSlashingsResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			return nil
		})),
	"/beacon/pool/voluntary_exits": newMetadata[structs.ListVoluntaryExitsResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*structs.ListVoluntaryExitsResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.ListVoluntaryExitsResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			return nil
		})),
	"/beacon/pool/bls_to_execution_changes": newMetadata[structs.BLSToExecutionChangesPoolResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*structs.BLSToExecutionChangesPoolResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.BLSToExecutionChangesPoolResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			return nil
		})),
	"/config/fork_schedule": newMetadata[structs.GetForkScheduleResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, lh interface{}) error {
			pResp, ok := p.(*structs.GetForkScheduleResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetForkScheduleResponse{}, p)
			}
			lhResp, ok := lh.(*structs.GetForkScheduleResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetForkScheduleResponse{}, lh)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lhResp.Data == nil {
				return errEmptyLighthouseData
			}
			// remove all forks with far-future epoch
			for i := len(pResp.Data) - 1; i >= 0; i-- {
				if pResp.Data[i].Epoch == fmt.Sprintf("%d", params.BeaconConfig().FarFutureEpoch) {
					pResp.Data = append(pResp.Data[:i], pResp.Data[i+1:]...)
				}
			}
			for i := len(lhResp.Data) - 1; i >= 0; i-- {
				if lhResp.Data[i].Epoch == fmt.Sprintf("%d", params.BeaconConfig().FarFutureEpoch) {
					lhResp.Data = append(lhResp.Data[:i], lhResp.Data[i+1:]...)
				}
			}
			return compareJSON(pResp, lhResp)
		})),
	"/config/deposit_contract": newMetadata[structs.GetDepositContractResponse](v1PathTemplate),
	"/debug/beacon/heads":      newMetadata[structs.GetForkChoiceHeadsV2Response](v2PathTemplate),
	"/node/identity": newMetadata[structs.GetIdentityResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*structs.GetIdentityResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetIdentityResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			return nil
		})),
	"/node/peers": newMetadata[structs.GetPeersResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*structs.GetPeersResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetPeersResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			return nil
		})),
	"/node/peer_count": newMetadata[structs.GetPeerCountResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, _ interface{}) error {
			pResp, ok := p.(*structs.GetPeerCountResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetPeerCountResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			return nil
		})),
	"/node/version": newMetadata[structs.GetVersionResponse](v1PathTemplate,
		withCustomEval(func(p interface{}, lh interface{}) error {
			pResp, ok := p.(*structs.GetVersionResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.ListAttestationsResponse{}, p)
			}
			lhResp, ok := lh.(*structs.GetVersionResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.ListAttestationsResponse{}, p)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if !strings.Contains(pResp.Data.Version, "Prysm") {
				return errors.New("version response does not contain Prysm client name")
			}
			if lhResp.Data == nil {
				return errEmptyLighthouseData
			}
			if !strings.Contains(lhResp.Data.Version, "Lighthouse") {
				return errors.New("version response does not contain Lighthouse client name")
			}
			return nil
		})),
	"/node/syncing": newMetadata[structs.SyncStatusResponse](v1PathTemplate),
	"/validator/duties/proposer/{param1}": newMetadata[structs.GetProposerDutiesResponse](v1PathTemplate,
		withParams(func(e primitives.Epoch) []string {
			return []string{fmt.Sprintf("%v", e)}
		}),
		withCustomEval(func(p interface{}, lh interface{}) error {
			pResp, ok := p.(*structs.GetProposerDutiesResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetProposerDutiesResponse{}, p)
			}
			lhResp, ok := lh.(*structs.GetProposerDutiesResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetProposerDutiesResponse{}, lh)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lhResp.Data == nil {
				return errEmptyLighthouseData
			}
			if lhResp.Data[0].Slot == "0" {
				// remove the first item from lighthouse data since lighthouse is returning a value despite no proposer
				// there is no proposer on slot 0 so prysm don't return anything for slot 0
				lhResp.Data = lhResp.Data[1:]
			}
			return compareJSON(pResp, lhResp)
		})),
	"/validator/duties/attester/{param1}": newMetadata[structs.GetAttesterDutiesResponse](v1PathTemplate,
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
		withCustomEval(func(p interface{}, lh interface{}) error {
			pResp, ok := p.(*structs.GetAttesterDutiesResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetAttesterDutiesResponse{}, p)
			}
			lhResp, ok := lh.(*structs.GetAttesterDutiesResponse)
			if !ok {
				return fmt.Errorf(msgWrongJson, &structs.GetAttesterDutiesResponse{}, lh)
			}
			if pResp.Data == nil {
				return errEmptyPrysmData
			}
			if lhResp.Data == nil {
				return errEmptyLighthouseData
			}
			return compareJSON(pResp, lhResp)
		})),
}
