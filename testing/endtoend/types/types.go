// Package types includes important structs used by end to end tests, such
// as a configuration type, an evaluator type, and more.
package types

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"google.golang.org/grpc"
)

// E2EConfig defines the struct for all configurations needed for E2E testing.
type E2EConfig struct {
	TestSync            bool
	UsePrysmShValidator bool
	UsePprof            bool
	TestDeposits        bool
	EpochsToRun         uint64
	TracingSinkEndpoint string
	Evaluators          []Evaluator
	BeaconFlags         []string
	ValidatorFlags      []string
}

// Evaluator defines the structure of the evaluators used to
// conduct the current beacon state during the E2E.
type Evaluator struct {
	Name       string
	Policy     func(currentEpoch types.Epoch) bool
	Evaluation func(conn ...*grpc.ClientConn) error // A variable amount of conns is allowed to be passed in for evaluations to check all nodes if needed.
}

// ComponentRunner defines an interface via which E2E component's configuration, execution and termination is managed.
type ComponentRunner interface {
	// Start starts a component.
	Start(ctx context.Context) error
	// Started checks whether an underlying component is started and ready to be queried.
	Started() <-chan struct{}
}
