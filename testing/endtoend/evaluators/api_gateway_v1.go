package evaluators

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/policies"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"google.golang.org/grpc"
)

type stateValidatorsResponseJson struct {
	Data []*validatorContainerJson `json:"data"`
}

type validatorContainerJson struct {
	Index     string         `json:"index"`
	Balance   string         `json:"balance"`
	Status    string         `json:"status"`
	Validator *validatorJson `json:"validator"`
}

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

// APIGatewayV1VerifyIntegrity of our API gateway for the Prysm v1 API.
// This ensures our gRPC HTTP gateway returns good data compared to some fixtures.
var APIGatewayV1VerifyIntegrity = e2etypes.Evaluator{
	Name:       "api_gateway_v1_verify_integrity_epoch_%d",
	Policy:     policies.OnEpoch(1),
	Evaluation: apiVerifyValidators,
}

func apiVerifyValidators(conns ...*grpc.ClientConn) error {
	count := len(conns)
	for i := 0; i < count; i++ {
		resp, err := http.Get(
			fmt.Sprintf("http://localhost:%d/eth/v1/beacon/states/head/validators?status=exited", e2e.TestParams.BeaconNodeRPCPort+i+30),
		)
		if err != nil {
			// Continue if the connection fails, regular flake.
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf("expected status code OK for beacon node %d, received %v with body %s", i, resp.StatusCode, body)
		}
		validators := &stateValidatorsResponseJson{}
		if err = json.NewDecoder(resp.Body).Decode(&validators); err != nil {
			return err
		}
		if len(validators.Data) != 0 {
			return fmt.Errorf("expected no exited validators to be returned from the API request for beacon node %d", i)
		}
		resp, err = http.Get(
			fmt.Sprintf("http://localhost:%d/eth/v1/beacon/states/head/validators?id=100&id=200", e2e.TestParams.BeaconNodeRPCPort+i+30),
		)
		if err != nil {
			// Continue if the connection fails, regular flake.
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf("expected status code OK for beacon node %d, received %v with body %s", i, resp.StatusCode, body)
		}
		validators = &stateValidatorsResponseJson{}
		if err = json.NewDecoder(resp.Body).Decode(&validators); err != nil {
			return err
		}
		if len(validators.Data) != 2 {
			return fmt.Errorf("expected 2 validators to be returned from the API request for beacon node %d", i)
		}
		if err = assertValidator(validators.Data[0]); err != nil {
			return errors.Wrapf(err, "incorrect validator data returned from the API request for beacon node %d", i)
		}
		if err = assertValidator(validators.Data[1]); err != nil {
			return errors.Wrapf(err, "incorrect validator data returned from the API request for beacon node %d", i)
		}

		if err = resp.Body.Close(); err != nil {
			return err
		}
		time.Sleep(connTimeDelay)
	}
	return nil
}

func assertValidator(v *validatorContainerJson) error {
	if v == nil {
		return errors.New("validator is nil")
	}
	if v.Index != "100" && v.Index != "200" {
		return fmt.Errorf("unexpected validator index '%s'", v.Index)
	}

	valid := false
	var field, expected, actual string

	switch v.Index {
	case "100":
		if v.Balance != "32000000000" {
			field = "Balance"
			expected = "32000000000"
			actual = v.Balance
			break
		}
		if v.Status != "active_ongoing" {
			field = "Status"
			expected = "active_ongoing"
			actual = v.Status
			break
		}
		if v.Validator == nil {
			return errors.New("validator is nil")
		}
		if v.Validator.PublicKey != "0x8931cd39ec3133b6ec91f26eec4de555cd7966086b1993dfe69c2b16e80adc62ce82d353b3356d8cc249e4e2d4254122" {
			field = "PublicKey"
			expected = "0x8931cd39ec3133b6ec91f26eec4de555cd7966086b1993dfe69c2b16e80adc62ce82d353b3356d8cc249e4e2d4254122"
			actual = v.Validator.PublicKey
			break
		}
		if v.Validator.WithdrawalCredentials != "0x00b5a389a138ec5069e430a91ec2884660fbb77a4bffdefd03f5e5769c2ba1a9" {
			field = "WithdrawalCredentials"
			expected = "0x00b5a389a138ec5069e430a91ec2884660fbb77a4bffdefd03f5e5769c2ba1a9"
			actual = v.Validator.WithdrawalCredentials
			break
		}
		if v.Validator.EffectiveBalance != "32000000000" {
			field = "EffectiveBalance"
			expected = "32000000000"
			actual = v.Validator.EffectiveBalance
			break
		}
		if v.Validator.Slashed {
			field = "Slashed"
			expected = "32000000000"
			actual = "true"
			break
		}
		if v.Validator.ActivationEligibilityEpoch != "0" {
			field = "ActivationEligibilityEpoch"
			expected = "0"
			actual = v.Validator.ActivationEligibilityEpoch
			break
		}
		if v.Validator.ActivationEpoch != "0" {
			field = "ActivationEpoch"
			expected = "0"
			actual = v.Validator.ActivationEpoch
			break
		}
		if v.Validator.ExitEpoch != "18446744073709551615" {
			field = "ExitEpoch"
			expected = "18446744073709551615"
			actual = v.Validator.ExitEpoch
			break
		}
		if v.Validator.WithdrawableEpoch != "18446744073709551615" {
			field = "WithdrawableEpoch"
			expected = "18446744073709551615"
			actual = v.Validator.WithdrawableEpoch
			break
		}
		valid = true
	case "200":
		if v.Balance != "32000000000" {
			field = "Balance"
			expected = "32000000000"
			actual = v.Balance
			break
		}
		if v.Status != "active_ongoing" {
			field = "Status"
			expected = "active_ongoing"
			actual = v.Status
			break
		}
		if v.Validator == nil {
			return errors.New("validator is nil")
		}
		if v.Validator.PublicKey != "0x8b4ff71ee947785f545c017bbb9ce84c3f6a90097368cf79663b2e11acc53e18e8f7159919784f4d28282cb39a7113f7" {
			field = "PublicKey"
			expected = "0x8b4ff71ee947785f545c017bbb9ce84c3f6a90097368cf79663b2e11acc53e18e8f7159919784f4d28282cb39a7113f7"
			actual = v.Validator.PublicKey
			break
		}
		if v.Validator.WithdrawalCredentials != "0x00b9ea0e53f64def81fe2e783a8bb02e57fb519f56a7224f93f2d37e1572417d" {
			field = "WithdrawalCredentials"
			expected = "0x00b9ea0e53f64def81fe2e783a8bb02e57fb519f56a7224f93f2d37e1572417d"
			actual = v.Validator.WithdrawalCredentials
			break
		}
		if v.Validator.EffectiveBalance != "32000000000" {
			field = "EffectiveBalance"
			expected = "32000000000"
			actual = v.Validator.EffectiveBalance
			break
		}
		if v.Validator.Slashed {
			field = "Slashed"
			expected = "32000000000"
			actual = "true"
			break
		}
		if v.Validator.ActivationEligibilityEpoch != "0" {
			field = "ActivationEligibilityEpoch"
			expected = "0"
			actual = v.Validator.ActivationEligibilityEpoch
			break
		}
		if v.Validator.ActivationEpoch != "0" {
			field = "ActivationEpoch"
			expected = "0"
			actual = v.Validator.ActivationEpoch
			break
		}
		if v.Validator.ExitEpoch != "18446744073709551615" {
			field = "ExitEpoch"
			expected = "18446744073709551615"
			actual = v.Validator.ExitEpoch
			break
		}
		if v.Validator.WithdrawableEpoch != "18446744073709551615" {
			field = "WithdrawableEpoch"
			expected = "18446744073709551615"
			actual = v.Validator.WithdrawableEpoch
			break
		}
		valid = true
	}

	if !valid {
		return fmt.Errorf("value of '%s' was expected to be '%s' but was '%s'", field, expected, actual)
	}
	return nil
}
