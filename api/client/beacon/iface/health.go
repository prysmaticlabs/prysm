package iface

import "context"

type HealthProvider interface {
	IsHealthy(ctx context.Context) bool
}

type HealthTracker interface {
	IsHealthy() bool
}
