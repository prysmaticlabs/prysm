package types

import (
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc"
)

// E2EConfig defines the struct for all configurations needed for E2E testing.
type E2EConfig struct {
	BeaconFlags     []string
	ValidatorFlags  []string
	NumBeaconNodes  uint64
	EpochsToRun     uint64
	TestSync        bool
	TestSlasher     bool
	Evaluators      []Evaluator
	ContractAddress common.Address
	TestPath        string
}

// Evaluator defines the structure of the evaluators used to
// conduct the current beacon state during the E2E.
type Evaluator struct {
	Name       string
	Policy     func(currentEpoch uint64) bool
	Evaluation func(conn ...*grpc.ClientConn) error
}

// BeaconNodeInfo contains the info of ports and other required information
// needed to communicate with the beacon node it represents.
type BeaconNodeInfo struct {
	ProcessID   int
	DataDir     string
	RPCPort     uint64
	MonitorPort uint64
	GRPCPort    uint64
	MultiAddr   string
}

type ValidatorClientInfo struct {
	ProcessID   int
	MonitorPort uint64
}
