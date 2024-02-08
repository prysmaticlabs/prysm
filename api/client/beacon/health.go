package beacon

import (
	"sync"
)

type NodeHealth struct {
	isHealthy bool
	HealthCh  chan bool
	sync.RWMutex
}

func NewNodeHealth() *NodeHealth {
	return &NodeHealth{
		isHealthy: true, // just default it to true
		HealthCh:  make(chan bool, 1),
	}
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
			n.HealthCh <- newStatus
		} else {
			n.Unlock()
		}
	}
}
