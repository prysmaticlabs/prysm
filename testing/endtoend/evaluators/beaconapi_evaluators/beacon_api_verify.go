package beaconapi_evaluators

import (
	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/policies"
	e2etypes "github.com/prysmaticlabs/prysm/v4/testing/endtoend/types"
	"google.golang.org/grpc"
)

// BeaconAPIMultiClientVerifyIntegrity tests Beacon API endpoints.
// It compares responses from Prysm and other beacon nodes such as Lighthouse.
// The evaluator is executed on every odd-numbered epoch.
var BeaconAPIMultiClientVerifyIntegrity = e2etypes.Evaluator{
	Name:       "beacon_api_multi-client_verify_integrity_epoch_%d",
	Policy:     policies.EveryNEpochs(1, 2),
	Evaluation: beaconAPIVerify,
}

const (
	v1PathTemplate = "http://localhost:%d/eth/v1"
	v2PathTemplate = "http://localhost:%d/eth/v2"
)

type apiComparisonFunc func(beaconNodeIdx int) error

func beaconAPIVerify(_ *e2etypes.EvaluationContext, conns ...*grpc.ClientConn) error {
	beacon := []apiComparisonFunc{
		withCompareBeaconAPIs,
	}
	for beaconNodeIdx := range conns {
		if err := runAPIComparisonFunctions(
			beaconNodeIdx,
			beacon...,
		); err != nil {
			return err
		}
	}
	return nil
}

func runAPIComparisonFunctions(beaconNodeIdx int, fs ...apiComparisonFunc) error {
	for _, f := range fs {
		if err := f(beaconNodeIdx); err != nil {
			return err
		}
	}
	return nil
}
