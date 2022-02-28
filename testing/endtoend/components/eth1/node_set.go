package eth1

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

// NodeSet represents a set of Eth1 nodes.
type NodeSet struct {
	e2etypes.ComponentRunner
	started      chan struct{}
	bootstrapEnr string
	minerEnrChan chan string
	keystorePath string
}

// NewNodeSet creates and returns a set of Eth1 nodes.
func NewNodeSet() *NodeSet {
	return &NodeSet{
		started:      make(chan struct{}, 1),
		minerEnrChan: make(chan string, 1),
	}
}

func (s *NodeSet) SetENR(bootstrapEnr string) {
	s.bootstrapEnr = bootstrapEnr
}

func (s *NodeSet) KeystorePath() string {
	return s.keystorePath
}

// Start starts all the beacon nodes in set.
func (s *NodeSet) Start(ctx context.Context) error {
	miner := NewMiner(s.bootstrapEnr, s.minerEnrChan)
	go func() {
		if err := miner.Start(ctx); err != nil {
			log.Error("Failed to start eth1 miner. Error message: " + err.Error())
		}
	}()

	// TODO: What to do here when miner does not start?
	minerEnr := <-s.minerEnrChan

	// Create Eth1 nodes. The number of nodes is the same as the number of beacon nodes.
	// We want each beacon node to connect to its own Eth1 node.
	// We start up one Eth1 node less than the beacon node count because the first
	// beacon node will connect to the already existing Eth1 miner.
	nodes := make([]e2etypes.ComponentRunner, e2e.TestParams.BeaconNodeCount-1)
	for i := 0; i < e2e.TestParams.BeaconNodeCount-1; i++ {
		// We start indexing nodes from 1 because the miner has an implicit 0 index.
		node := NewNode(i+1, minerEnr)
		nodes[i] = node
	}

	// Wait for all nodes to finish their job (blocking).
	// Once nodes are ready passed in handler function will be called.
	return helpers.WaitOnNodes(ctx, append(nodes, miner), func() {
		// All nodes started, close channel, so that all services waiting on a set, can proceed.
		s.keystorePath = miner.KeystorePath()
		close(s.started)
	})
}

// Started checks whether beacon node set is started and all nodes are ready to be queried.
func (s *NodeSet) Started() <-chan struct{} {
	return s.started
}
