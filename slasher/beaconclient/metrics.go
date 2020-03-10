package beaconclient

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	slasherNumAttestationsReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slasher_num_attestations_received",
		Help: "The # of attestations received by slasher",
	})
)
