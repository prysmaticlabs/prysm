package beaconapi_evaluators

import (
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/policies"
	e2etypes "github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
	"google.golang.org/grpc"
)

// APIMiddlewareVerifyIntegrity tests our API Middleware for the official Ethereum API.
// This ensures our API Middleware returns good data compared to gRPC.
var APIMiddlewareVerifyIntegrity = e2etypes.Evaluator{
	Name:       "api_middleware_verify_integrity_epoch_%d",
	Policy:     policies.AllEpochs,
	Evaluation: apiMiddlewareVerify,
}

const (
	v1MiddlewarePathTemplate = "http://localhost:%d/eth/v1"
	v2MiddlewarePathTemplate = "http://localhost:%d/eth/v2"
)

type apiComparisonFunc func(beaconNodeIdx int, conn *grpc.ClientConn) error

func apiMiddlewareVerify(conns ...*grpc.ClientConn) error {
	beacon := []apiComparisonFunc{
		withCompareBeaconBlocks,
		withCompareValidatorsEth,
		withCompareSyncCommittee,
		withCompareBlockAttestations,
	}
	validator := []apiComparisonFunc{
		withCompareAttesterDuties,
	}
	node := []apiComparisonFunc{
		withCompareNodeMetaData,
	}
	comparisons := append(beacon, validator...)
	comparisons = append(comparisons, node...)
	for beaconNodeIdx, conn := range conns {
		if err := runAPIComparisonFunctions(
			beaconNodeIdx,
			conn,
			comparisons...,
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
