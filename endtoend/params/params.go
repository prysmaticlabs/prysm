// Package params defines all custom parameter configurations
// for running end to end tests.
package params

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/common"
)

// Params struct defines the parameters needed for running E2E tests to properly handle test sharding.
type Params struct {
	TestPath              string
	LogPath               string
	TestShardIndex        int
	BeaconNodeCount       int
	Eth1RPCPort           int
	ContractAddress       common.Address
	BootNodePort          int
	BootNodeENR           string
	BeaconNodeRPCPort     int
	BeaconNodeMetricsPort int
	ValidatorMetricsPort  int
	SlasherRPCPort        int
	SlasherMetricsPort    int
}

// TestParams is the globally accessible var for getting config elements.
var TestParams *Params

// BootNodeLogFileName is the file name used for the beacon chain node logs.
var BootNodeLogFileName = "bootnode.log"

// BeaconNodeLogFileName is the file name used for the beacon chain node logs.
var BeaconNodeLogFileName = "beacon-%d.log"

// SlasherLogFileName is the file name used for the slasher client logs.
var SlasherLogFileName = "slasher-%d.log"

// ValidatorLogFileName is the file name used for the validator client logs.
var ValidatorLogFileName = "vals-%d.log"

// LongRunningBeaconCount is a global constant for the count of beacon nodes of long running E2E.
var LongRunningBeaconCount = 2

// StandardBeaconCount is a global constant for the count of beacon nodes of standard E2E tests.
var StandardBeaconCount = 2

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

	TestParams = &Params{
		TestPath:              path.Join(testPath, fmt.Sprintf("shard-%d", testIndex)),
		LogPath:               logPath,
		TestShardIndex:        testIndex,
		BeaconNodeCount:       beaconNodeCount,
		Eth1RPCPort:           3100 + testIndex*100, // Multiplying 100 here so the test index doesn't conflict with the other node ports.
		BootNodePort:          4100 + testIndex*100,
		BeaconNodeRPCPort:     4150 + testIndex*100,
		BeaconNodeMetricsPort: 5100 + testIndex*100,
		ValidatorMetricsPort:  6100 + testIndex*100,
		SlasherRPCPort:        7100 + testIndex*100,
		SlasherMetricsPort:    8100 + testIndex*100,
	}
	return nil
}
