package beacon

import (
	"sync"
)

type NodeHealth struct {
	isHealthy bool
	healthCh  chan bool
	sync.RWMutex
}

func NewNodeHealth(initialStatus bool) *NodeHealth {
	return &NodeHealth{
		isHealthy: initialStatus, // just default it to true
		healthCh:  make(chan bool, 1),
	}
}

// HealthUpdates provides a read-only channel for health updates.
func (n *NodeHealth) HealthUpdates() <-chan bool {
	return n.healthCh
}

func (n *NodeHealth) IsHealthy() bool {
	n.RLock()
	defer n.RUnlock()
	return n.isHealthy
}

func (n *NodeHealth) UpdateNodeHealth(newStatus bool) {
	n.RLock()
	isStatusChanged := newStatus != n.isHealthy
	n.RUnlock()

	if isStatusChanged {
		n.Lock()
		// Double-check the condition to ensure it hasn't changed since the first check.
		if newStatus != n.isHealthy {
			n.isHealthy = newStatus
			n.Unlock() // It's better to unlock as soon as the protected section is over.
			n.healthCh <- newStatus
		} else {
			n.Unlock()
		}
	}
}
