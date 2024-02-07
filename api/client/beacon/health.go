package beacon

import (
	"sync"
)

type NodeHealth struct {
	isHealthy     bool
	HealthCh      chan bool
	isHealthyLock sync.RWMutex
}

func NewNodeHealthTracker() *NodeHealth {
	return &NodeHealth{
		isHealthy: true, // just default it to true
		HealthCh:  make(chan bool, 1),
	}
}

func (n *NodeHealth) IsHealthy() bool {
	n.isHealthyLock.RLock()
	defer n.isHealthyLock.RUnlock()
	return n.isHealthy
}

func (n *NodeHealth) UpdateNodeHealth(newStatus bool) {
	n.isHealthyLock.RLock()
	isStatusChanged := newStatus != n.isHealthy
	n.isHealthyLock.RUnlock()

	if isStatusChanged {
		n.isHealthyLock.Lock()
		// Double-check the condition to ensure it hasn't changed since the first check.
		if newStatus != n.isHealthy {
			n.isHealthy = newStatus
			n.isHealthyLock.Unlock() // It's better to unlock as soon as the protected section is over.
			n.HealthCh <- newStatus
		} else {
			n.isHealthyLock.Unlock()
		}
	}
}
