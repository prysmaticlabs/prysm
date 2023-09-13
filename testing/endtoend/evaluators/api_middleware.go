package evaluators

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/proto/eth/service"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/helpers"
	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/policies"
	e2etypes "github.com/prysmaticlabs/prysm/v4/testing/endtoend/types"
	"google.golang.org/grpc"
)

type validatorJson struct {
	PublicKey                  string `json:"pubkey"`
	WithdrawalCredentials      string `json:"withdrawal_credentials"`
	EffectiveBalance           string `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch string `json:"activation_eligibility_epoch"`
	ActivationEpoch            string `json:"activation_epoch"`
	ExitEpoch                  string `json:"exit_epoch"`
	WithdrawableEpoch          string `json:"withdrawable_epoch"`
}
type validatorContainerJson struct {
	Index     string         `json:"index"`
	Balance   string         `json:"balance"`
	Status    string         `json:"status"`
	Validator *validatorJson `json:"validator"`
}

// APIMiddlewareVerifyIntegrity tests our API Middleware for the official Ethereum API.
// This ensures our API Middleware returns good data compared to gRPC.
var APIMiddlewareVerifyIntegrity = e2etypes.Evaluator{
	Name:       "api_middleware_verify_integrity_epoch_%d",
	Policy:     policies.OnEpoch(helpers.AltairE2EForkEpoch),
	Evaluation: apiMiddlewareVerify,
}

const (
	v1MiddlewarePathTemplate = "http://localhost:%d/eth/v1"
)

func apiMiddlewareVerify(_ *e2etypes.EvaluationContext, conns ...*grpc.ClientConn) error {
	for beaconNodeIdx, conn := range conns {
		if err := runAPIComparisonFunctions(
			beaconNodeIdx,
			conn,
			withCompareSyncCommittee,
			withCompareAttesterDuties,
		); err != nil {
			return err
		}
	}
	return nil
}

func withCompareSyncCommittee(beaconNodeIdx int, conn *grpc.ClientConn) error {
	type syncCommitteeValidatorsJson struct {
		Validators          []string   `json:"validators"`
		ValidatorAggregates [][]string `json:"validator_aggregates"`
	}
	type syncCommitteesResponseJson struct {
		Data *syncCommitteeValidatorsJson `json:"data"`
	}
	ctx := context.Background()
	beaconClient := service.NewBeaconChainClient(conn)
	resp, err := beaconClient.ListSyncCommittees(ctx, &ethpbv2.StateSyncCommitteesRequest{
		StateId: []byte("head"),
	})
	if err != nil {
		return err
	}
	respJSON := &syncCommitteesResponseJson{}
	if err := doMiddlewareJSONGetRequestV1(
		"/beacon/states/head/sync_committees",
		beaconNodeIdx,
		respJSON,
	); err != nil {
		return err
	}
	if len(respJSON.Data.Validators) != len(resp.Data.Validators) {
		return fmt.Errorf(
			"API Middleware number of validators %d does not match gRPC %d",
			len(respJSON.Data.Validators),
			len(resp.Data.Validators),
		)
	}
	if len(respJSON.Data.ValidatorAggregates) != len(resp.Data.ValidatorAggregates) {
		return fmt.Errorf(
			"API Middleware number of validator aggregates %d does not match gRPC %d",
			len(respJSON.Data.ValidatorAggregates),
			len(resp.Data.ValidatorAggregates),
		)
	}
	return nil
}

func withCompareAttesterDuties(beaconNodeIdx int, conn *grpc.ClientConn) error {
	type attesterDutyJson struct {
		Pubkey                  string `json:"pubkey" hex:"true"`
		ValidatorIndex          string `json:"validator_index"`
		CommitteeIndex          string `json:"committee_index"`
		CommitteeLength         string `json:"committee_length"`
		CommitteesAtSlot        string `json:"committees_at_slot"`
		ValidatorCommitteeIndex string `json:"validator_committee_index"`
		Slot                    string `json:"slot"`
	}
	type attesterDutiesResponseJson struct {
		DependentRoot string              `json:"dependent_root" hex:"true"`
		Data          []*attesterDutyJson `json:"data"`
	}
	ctx := context.Background()
	validatorClient := service.NewBeaconValidatorClient(conn)
	resp, err := validatorClient.GetAttesterDuties(ctx, &ethpbv1.AttesterDutiesRequest{
		Epoch: helpers.AltairE2EForkEpoch,
		Index: []primitives.ValidatorIndex{0},
	})
	if err != nil {
		return err
	}
	// We post a top-level array, not an object, as per the spec.
	reqJSON := []string{"0"}
	respJSON := &attesterDutiesResponseJson{}
	if err := doMiddlewareJSONPostRequestV1(
		"/validator/duties/attester/"+strconv.Itoa(helpers.AltairE2EForkEpoch),
		beaconNodeIdx,
		reqJSON,
		respJSON,
	); err != nil {
		return err
	}
	if respJSON.DependentRoot != hexutil.Encode(resp.DependentRoot) {
		return buildFieldError("DependentRoot", string(resp.DependentRoot), respJSON.DependentRoot)
	}
	if len(respJSON.Data) != len(resp.Data) {
		return fmt.Errorf(
			"API Middleware number of duties %d does not match gRPC %d",
			len(respJSON.Data),
			len(resp.Data),
		)
	}
	return nil
}

func doMiddlewareJSONGetRequestV1(requestPath string, beaconNodeIdx int, dst interface{}) error {
	basePath := fmt.Sprintf(v1MiddlewarePathTemplate, params.TestParams.Ports.PrysmBeaconNodeGatewayPort+beaconNodeIdx)
	httpResp, err := http.Get(
		basePath + requestPath,
	)
	if err != nil {
		return err
	}
	return json.NewDecoder(httpResp.Body).Decode(&dst)
}

func doMiddlewareJSONPostRequestV1(requestPath string, beaconNodeIdx int, postData, dst interface{}) error {
	b, err := json.Marshal(postData)
	if err != nil {
		return err
	}
	basePath := fmt.Sprintf(v1MiddlewarePathTemplate, params.TestParams.Ports.PrysmBeaconNodeGatewayPort+beaconNodeIdx)
	httpResp, err := http.Post(
		basePath+requestPath,
		"application/json",
		bytes.NewBuffer(b),
	)
	if err != nil {
		return err
	}
	return json.NewDecoder(httpResp.Body).Decode(&dst)
}

func buildFieldError(field, expected, actual string) error {
	return fmt.Errorf("value of '%s' was expected to be '%s' but was '%s'", field, expected, actual)
}
