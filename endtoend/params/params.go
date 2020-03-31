package params

import (
	"errors"
	"os"
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
	BeaconNodeRPCPort     int
	BeaconNodeMetricsPort int
	ValidatorMetricsPort  int
	SlasherRPCPort        int
	SlasherMetricsPort    int
}

// TestParams is the globally accessible var for getting config elements.
var TestParams *Params

// BeaconNodeLogFileName is the file name used for the beacon chain node logs.
var BeaconNodeLogFileName = "beacon-%d.log"

// SlasherLogFileName is the file name used for the slasher client logs.
var SlasherLogFileName = "slasher-%d.log"

// ValidatorLogFileName is the file name used for the validator client logs.
var ValidatorLogFileName = "vals-%d.log"

// Init initializes the E2E config, properly handling test sharding.
func Init(beaconNodeCount int) error {
	testPath := bazel.TestTmpDir()
	logPath, ok := os.LookupEnv("TEST_UNDECLARED_OUTPUTS_DIR")
	if !ok {
		return errors.New("expected TEST_UNDECLARED_OUTPUTS_DIR to be defined")
	}
	testIndexStr, ok := os.LookupEnv("TEST_SHARD_INDEX")
	if !ok {
		testIndexStr = "8"
	}
	testIndex, err := strconv.Atoi(testIndexStr)
	if err != nil {
		return err
	}

	TestParams = &Params{
		TestPath:              testPath,
		LogPath:               logPath,
		TestShardIndex:        testIndex,
		BeaconNodeCount:       beaconNodeCount,
		Eth1RPCPort:           3000 + testIndex*100, // Multiplying 100 here so the test index doesn't conflict with the other node ports.
		BeaconNodeRPCPort:     4000 + testIndex*100,
		BeaconNodeMetricsPort: 5000 + testIndex*100,
		ValidatorMetricsPort:  6000 + testIndex*100,
		SlasherRPCPort:        7000 + testIndex*100,
		SlasherMetricsPort:    8000 + testIndex*100,
	}
	return nil
}
