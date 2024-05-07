package beacon

import (
	"context"
	"sync"

	"github.com/prysmaticlabs/prysm/v5/api/client/beacon/iface"
)

type NodeHealthTracker struct {
	isHealthy  *bool
	healthChan chan bool
	node       iface.HealthNode
	sync.RWMutex
}

func NewNodeHealthTracker(node iface.HealthNode) *NodeHealthTracker {
	return &NodeHealthTracker{
		node:       node,
		healthChan: make(chan bool, 1),
	}
}

// HealthUpdates provides a read-only channel for health updates.
func (n *NodeHealthTracker) HealthUpdates() <-chan bool {
	return n.healthChan
}

func (n *NodeHealthTracker) IsHealthy() bool {
	n.RLock()
	defer n.RUnlock()
	if n.isHealthy == nil {
		return false
	}
	return *n.isHealthy
}

func (n *NodeHealthTracker) CheckHealth(ctx context.Context) bool {
	n.RLock()
	newStatus := n.node.IsHealthy(ctx)
	if n.isHealthy == nil {
		n.isHealthy = &newStatus
	}
	isStatusChanged := newStatus != *n.isHealthy
	n.RUnlock()

	if isStatusChanged {
		n.Lock()
		// Double-check the condition to ensure it hasn't changed since the first check.
		n.isHealthy = &newStatus
		n.Unlock() // It's better to unlock as soon as the protected section is over.
		n.healthChan <- newStatus
	}
	return newStatus
}
