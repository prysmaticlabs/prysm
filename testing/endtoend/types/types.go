// Package types includes important structs used by end to end tests, such
// as a configuration type, an evaluator type, and more.
package types

import (
	"context"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"google.golang.org/grpc"
)

type E2EConfigOpt func(*E2EConfig)

func WithEpochs(e uint64) E2EConfigOpt {
	return func(cfg *E2EConfig) {
		cfg.EpochsToRun = e
	}
}

func WithRemoteSigner() E2EConfigOpt {
	return func(cfg *E2EConfig) {
		cfg.UseWeb3RemoteSigner = true
	}
}

func WithCheckpointSync() E2EConfigOpt {
	return func(cfg *E2EConfig) {
		cfg.TestCheckpointSync = true
	}
}

// E2EConfig defines the struct for all configurations needed for E2E testing.
type E2EConfig struct {
	EvalInterceptor         func(uint64, []*grpc.ClientConn) bool
	TracingSinkEndpoint     string
	PeerIDs                 []string
	ValidatorFlags          []string
	BeaconFlags             []string
	Evaluators              []Evaluator
	EpochsToRun             uint64
	ExtraEpochs             uint64
	Seed                    int64
	UsePprof                bool
	UseValidatorCrossClient bool
	UseFixedPeerIDs         bool
	TestDeposits            bool
	UseWeb3RemoteSigner     bool
	TestCheckpointSync      bool
	UsePrysmShValidator     bool
	TestFeature             bool
	TestSync                bool
}

// Evaluator defines the structure of the evaluators used to
// conduct the current beacon state during the E2E.
type Evaluator struct {
	Policy     func(currentEpoch types.Epoch) bool
	Evaluation func(conn ...*grpc.ClientConn) error // A variable amount of conns is allowed to be passed in for evaluations to check all nodes if needed.
	Name       string
}

// ComponentRunner defines an interface via which E2E component's configuration, execution and termination is managed.
type ComponentRunner interface {
	// Start starts a component.
	Start(ctx context.Context) error
	// Started checks whether an underlying component is started and ready to be queried.
	Started() <-chan struct{}
	// Pause pauses a component.
	Pause() error
	// Resume resumes a component.
	Resume() error
	// Stop stops a component.
	Stop() error
}

type MultipleComponentRunners interface {
	ComponentRunner
	// ComponentAtIndex returns the component at index
	ComponentAtIndex(i int) (ComponentRunner, error)
	// PauseAtIndex pauses the grouped component element at the desired index.
	PauseAtIndex(i int) error
	// ResumeAtIndex resumes the grouped component element at the desired index.
	ResumeAtIndex(i int) error
	// StopAtIndex stops the grouped component element at the desired index.
	StopAtIndex(i int) error
}

type EngineProxy interface {
	ComponentRunner
	// AddRequestInterceptor adds in a json-rpc request interceptor.
	AddRequestInterceptor(rpcMethodName string, responseGen func() interface{}, trigger func() bool)
	// RemoveRequestInterceptor removes the request interceptor for the provided method.
	RemoveRequestInterceptor(rpcMethodName string)
	// ReleaseBackedUpRequests releases backed up http requests.
	ReleaseBackedUpRequests(rpcMethodName string)
}

// BeaconNodeSet defines an interface for an object that fulfills the duties
// of a group of beacon nodes.
type BeaconNodeSet interface {
	ComponentRunner
	// SetENR provides the relevant bootnode's enr to the beacon nodes.
	SetENR(enr string)
}
