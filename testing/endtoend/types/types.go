// Package types includes important structs used by end to end tests, such
// as a configuration type, an evaluator type, and more.
package types

import (
	"context"
	"fmt"
	"path"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/io/file"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	types "github.com/prysmaticlabs/eth2-types"
	"google.golang.org/grpc"
)

// E2EConfig defines the struct for all configurations needed for E2E testing.
type E2EConfig struct {
	TestSync                bool
	TestFeature             bool
	UsePrysmShValidator     bool
	UsePprof                bool
	UseWeb3RemoteSigner     bool
	TestDeposits            bool
	UseFixedPeerIDs         bool
	UseValidatorCrossClient bool
	EpochsToRun             uint64
	Seed                    int64
	TracingSinkEndpoint     string
	Evaluators              []Evaluator
	BeaconFlags             []string
	ValidatorFlags          []string
	PeerIDs                 []string
	BeaconChainConfig       *params.BeaconChainConfig
}

// BeaconChainConfigPath determines the canonical path to the yaml-encoded BeaconChainConfig
// written by WriteBeaconChainConfig. Used by components to load a non-standard config in tests.
func (cfg *E2EConfig) BeaconChainConfigPath() string {
	fname := fmt.Sprintf("beacon-chain-config_%s.yaml", cfg.BeaconChainConfig.ConfigName)
	return path.Join(e2e.TestParams.LogPath, fname)
}

// WriteBeaconChainConfig writes the yaml encoding of the BeaconChainConfig struct member
// to a file at the path specified by BeaconChainConfigPath.
func (cfg *E2EConfig) WriteBeaconChainConfig() error {
	return file.WriteFile(cfg.BeaconChainConfigPath(), params.ConfigToYaml(cfg.BeaconChainConfig))
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
