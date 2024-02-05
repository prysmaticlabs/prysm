package beacon

type NodeHealth struct {
	IsHealthy bool
	HealthCh  chan bool
}
