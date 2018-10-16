package metric

import (
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

var (
	messagesCompleted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "p2p_message_sended_total",
			Help: "Count of messages sended.",
		},
	)
	sendLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "p2p_message_sended_latency_seconds",
			Help:    "Latency of messages sended.",
			Buckets: []float64{.01, .03, .1, .3, 1, 3, 10, 30, 100, 300},
		},
	)
	messageSize = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "p2p_message_size_bytes",
			Help:    "Size of received messages.",
			Buckets: prometheus.ExponentialBuckets(32, 32, 6),
		},
	)

	p2pInit sync.Once
)

func New() p2p.Adapter {
	p2pInit.Do(func() {
		prometheus.MustRegister(messagesCompleted)
		prometheus.MustRegister(sendLatency)
		prometheus.MustRegister(messageSize)
	})

	return func(next p2p.Handler) p2p.Handler {
		return func(msg p2p.Message) {
			start := time.Now()
			messageSize.Observe(float64(proto.Size(msg.Data)))
			next(msg)
			sendLatency.Observe(float64(time.Since(start)))
			messagesCompleted.Inc()
		}
	}
}
