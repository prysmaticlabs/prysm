// Package params defines all custom parameter configurations
// for running end to end tests.
package params

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/common"
)

// params struct defines the parameters needed for running E2E tests to properly handle test sharding.
type params struct {
	TestPath              string
	LogPath               string
	TestShardIndex        int
	BeaconNodeCount       int
	Eth1RPCPort           int
	ContractAddress       common.Address
	BootNodePort          int
	BeaconNodeRPCPort     int
	BeaconNodeOAPIPort    int
	BeaconNodeMetricsPort int
	ValidatorMetricsPort  int
	ValidatorGatewayPort  int
}

// TestParams is the globally accessible var for getting config elements.
var TestParams *params

// BootNodeLogFileName is the file name used for the beacon chain node logs.
var BootNodeLogFileName = "bootnode.log"

// TracingRequestSinkFileName is the file name for writing raw trace requests.
var TracingRequestSinkFileName = "tracing-http-requests.log.gz"

// BeaconNodeLogFileName is the file name used for the beacon chain node logs.
var BeaconNodeLogFileName = "beacon-%d.log"

// ValidatorLogFileName is the file name used for the validator client logs.
var ValidatorLogFileName = "vals-%d.log"

// StandardBeaconCount is a global constant for the count of beacon nodes of standard E2E tests.
var StandardBeaconCount = 2

// DepositCount is the amount of deposits E2E makes on a separate validator client.
var DepositCount = uint64(64)

// Init initializes the E2E config, properly handling test sharding.
func Init(beaconNodeCount int) error {
	testPath := bazel.TestTmpDir()
	logPath, ok := os.LookupEnv("TEST_UNDECLARED_OUTPUTS_DIR")
	if !ok {
		return errors.New("expected TEST_UNDECLARED_OUTPUTS_DIR to be defined")
	}
	testIndexStr, ok := os.LookupEnv("TEST_SHARD_INDEX")
	if !ok {
		testIndexStr = "0"
	}
	testIndex, err := strconv.Atoi(testIndexStr)
	if err != nil {
		return err
	}
	testPath = filepath.Join(testPath, fmt.Sprintf("shard-%d", testIndex))

	TestParams = &params{
		TestPath:              testPath,
		LogPath:               logPath,
		TestShardIndex:        testIndex,
		BeaconNodeCount:       beaconNodeCount,
		Eth1RPCPort:           3100 + testIndex*100, // Multiplying 100 here so the test index doesn't conflict with the other node ports.
		BootNodePort:          4100 + testIndex*100,
		BeaconNodeRPCPort:     4150 + testIndex*100,
		BeaconNodeMetricsPort: 5100 + testIndex*100,
		ValidatorMetricsPort:  6100 + testIndex*100,
		ValidatorGatewayPort:  7150 + testIndex*100,
	}
	return nil
}
