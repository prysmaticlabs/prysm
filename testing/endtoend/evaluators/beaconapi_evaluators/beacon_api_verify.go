package beaconapi_evaluators

import (
	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/policies"
	e2etypes "github.com/prysmaticlabs/prysm/v4/testing/endtoend/types"
	"google.golang.org/grpc"
)

// BeaconAPIMultiClientVerifyIntegrity tests our API Middleware responses to other beacon nodes such as lighthouse.
var BeaconAPIMultiClientVerifyIntegrity = e2etypes.Evaluator{
	Name:       "beacon_api_multi-client_verify_integrity_epoch_%d",
	Policy:     policies.AfterNthEpoch(0),
	Evaluation: beaconAPIVerify,
}

const (
	v1MiddlewarePathTemplate = "http://localhost:%d/eth/v1"
	v2MiddlewarePathTemplate = "http://localhost:%d/eth/v2"
)

type apiComparisonFunc func(beaconNodeIdx int, conn *grpc.ClientConn) error

func beaconAPIVerify(_ *e2etypes.EvaluationContext, conns ...*grpc.ClientConn) error {
	beacon := []apiComparisonFunc{
		withCompareBeaconAPIs,
	}
	for beaconNodeIdx, conn := range conns {
		if err := runAPIComparisonFunctions(
			beaconNodeIdx,
			conn,
			beacon...,
		); err != nil {
			return err
		}
	}
	return nil
}

func runAPIComparisonFunctions(beaconNodeIdx int, conn *grpc.ClientConn, fs ...apiComparisonFunc) error {
	for _, f := range fs {
		if err := f(beaconNodeIdx, conn); err != nil {
			return err
		}
	}
	return nil
}
