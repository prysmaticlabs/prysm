package beacon

import (
	"context"
	"sync"
	"time"

	"github.com/prysmaticlabs/prysm/v5/api/client/beacon/iface"
	"github.com/prysmaticlabs/prysm/v5/config/params"
)

type NodeHealthTracker struct {
	isHealthy bool
	node      iface.HealthNode
	sync.RWMutex
}

func NewNodeHealthTracker(ctx context.Context, node iface.HealthNode) *NodeHealthTracker {
	tracker := &NodeHealthTracker{
		node:      node,
		isHealthy: true,
	}
	log.Info("Starting health check routine. Health check will be performed every 12 seconds.")
	ticker := time.NewTicker(time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	go func() {
		for range ticker.C {
			tracker.Lock()
			tracker.isHealthy = tracker.node.IsHealthy(ctx)
			tracker.Unlock()
		}
		log.Info("Stopping health check routine")
	}()
	return tracker
}

func (n *NodeHealthTracker) IsHealthy() bool {
	n.RLock()
	defer n.RUnlock()
	return n.isHealthy
}
