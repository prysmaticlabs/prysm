// Package metric contain some prometheus collectors for p2p services.
package metric

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

var (
	messagesCompleted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_sent_total",
			Help: "Count of messages sent.",
		},
		[]string{"message"},
	)
	sendLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "p2p_message_sent_latency_seconds",
			Help:    "Latency of messages sent.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
		},
		[]string{"message"},
	)
	messageSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "p2p_message_received_bytes",
			Help:    "Size of received messages.",
			Buckets: prometheus.ExponentialBuckets(32, 32, 6),
		},
		[]string{"message"},
	)
)

// New create and initialize a metric adapter for the p2p service.
func New() p2p.Adapter {
	return func(next p2p.Handler) p2p.Handler {
		return func(msg p2p.Message) {
			start := time.Now()
			messageName := fmt.Sprintf("%T", msg.Data)

			messageSize.WithLabelValues(messageName).Observe(float64(proto.Size(msg.Data)))
			next(msg)
			sendLatency.WithLabelValues(messageName).Observe(time.Since(start).Seconds())
			messagesCompleted.WithLabelValues(messageName).Inc()
		}
	}
}
