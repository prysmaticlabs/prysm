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
	node      iface.HealthProvider
	sync.RWMutex
}

func NewNodeHealthTracker(ctx context.Context, node iface.HealthProvider) *NodeHealthTracker {
	tracker := &NodeHealthTracker{
		node:      node,
		isHealthy: true,
	}
	log.Infof("Starting health check routine. Health check will be performed every %d seconds", params.BeaconConfig().SecondsPerSlot)
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
