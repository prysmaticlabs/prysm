package params

import (
	"errors"
	"os"
	"strconv"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/common"
)

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
}

var TestParams *Params

var BeaconNodeLogFileName = "beacon-%d.log"

var SlasherLogFileName = "slasher-%d.log"

var ValidatorLogFileName = "vals-%d.log"

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
		TestPath:              testPath,
		LogPath:               logPath,
		TestShardIndex:        testIndex,
		BeaconNodeCount:       beaconNodeCount,
		Eth1RPCPort:           1000 + testIndex*100, //Multiplying 100 here so the test index doesn't conflict with the other node ports.
		BeaconNodeRPCPort:     2000 + testIndex*100,
		BeaconNodeMetricsPort: 3000 + testIndex*100,
		ValidatorMetricsPort:  4000 + testIndex*100,
	}
	return nil
}
