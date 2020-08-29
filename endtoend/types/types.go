// Package types includes important structs used by end to end tests, such
// as a configuration type, an evaluator type, and more.
package types

import (
	"google.golang.org/grpc"
)

// E2EConfig defines the struct for all configurations needed for E2E testing.
type E2EConfig struct {
	BeaconFlags    []string
	ValidatorFlags []string
	EpochsToRun    uint64
	TestSync       bool
	TestSlasher    bool
	TestDeposits   bool
	UsePprof       bool
	Evaluators     []Evaluator
}

// Evaluator defines the structure of the evaluators used to
// conduct the current beacon state during the E2E.
type Evaluator struct {
	Name       string
	Policy     func(currentEpoch uint64) bool
	Evaluation func(conn ...*grpc.ClientConn) error // A variable amount of conns is allowed to be passed in for evaluations to check all nodes if needed.
}

// BeaconNodeInfo contains the info of ports and other required information
// needed to communicate with the beacon node it represents.
type BeaconNodeInfo struct {
	ProcessID int
	MultiAddr string
}
