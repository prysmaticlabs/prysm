// Package metric contain some prometheus collectors for p2p services.
package metric

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

var log = logrus.WithField("prefix", "prometheus")

var (
	messagesCompleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_message_sended_total",
			Help: "Count of messages sended.",
		},
		[]string{"message"},
	)
	sendLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "p2p_message_sended_latency_seconds",
			Help:    "Latency of messages sent.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
		},
		[]string{"message"},
	)
	messageSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "p2p_message_size_bytes",
			Help:    "Size of received messages.",
			Buckets: prometheus.ExponentialBuckets(32, 32, 6),
		},
		[]string{"message"},
	)

	p2pInit sync.Once
)

// New create and initialize a metric adapter for the p2p service.
func New() p2p.Adapter {
	p2pInit.Do(func() {
		prometheus.MustRegister(messagesCompleted)
		prometheus.MustRegister(sendLatency)
		prometheus.MustRegister(messageSize)
	})

	return func(next p2p.Handler) p2p.Handler {
		return func(msg p2p.Message) {
			start := time.Now()
			messageName := fmt.Sprintf("%T", msg.Data)

			messageSize.WithLabelValues(messageName).Observe(float64(proto.Size(msg.Data)))
			next(msg)
			time.Sleep(time.Duration(rand.Int63() % int64(time.Second)))
			sendLatency.WithLabelValues(messageName).Observe(time.Since(start).Seconds())
			messagesCompleted.WithLabelValues(messageName).Inc()
		}
	}
}
