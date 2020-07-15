package sync

import (
	"reflect"
	"sync"

	"github.com/kevinms/leakybucket-go"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/sirupsen/logrus"
)

const defaultBurstLimit = 5

type limiter struct {
	limiterMap map[string]*leakybucket.Collector
	p2p        p2p.P2P
	sync.RWMutex
}

// Instantiates a multi-rpc protocol rate limiter, providing
// separate collectors for each topic.
func newRateLimiter(p2pProvider p2p.P2P) *limiter {
	// add encoding suffix
	addEncoding := func(topic string) string {
		return topic + p2pProvider.Encoding().ProtocolSuffix()
	}
	// Initialize block limits.
	allowedBlocksPerSecond := float64(flags.Get().BlockBatchLimit)
	allowedBlocksBurst := int64(flags.Get().BlockBatchLimitBurstFactor * flags.Get().BlockBatchLimit)

	// Set topic map for all rpc topics.
	topicMap := make(map[string]*leakybucket.Collector, len(p2p.RPCTopicMappings))
	// Goodbye Message
	topicMap[addEncoding(p2p.RPCGoodByeTopic)] = leakybucket.NewCollector(1, 1, false /* deleteEmptyBuckets */)
	// Metadata Message
	topicMap[addEncoding(p2p.RPCMetaDataTopic)] = leakybucket.NewCollector(1, defaultBurstLimit, false /* deleteEmptyBuckets */)
	// Ping Message
	topicMap[addEncoding(p2p.RPCPingTopic)] = leakybucket.NewCollector(1, defaultBurstLimit, false /* deleteEmptyBuckets */)
	// Status Message
	topicMap[addEncoding(p2p.RPCStatusTopic)] = leakybucket.NewCollector(1, defaultBurstLimit, false /* deleteEmptyBuckets */)

	// Use a single collector for block requests
	blockCollector := leakybucket.NewCollector(allowedBlocksPerSecond, allowedBlocksBurst, false /* deleteEmptyBuckets */)

	// BlocksByRoots requests
	topicMap[addEncoding(p2p.RPCBlocksByRootTopic)] = blockCollector

	// BlockByRange requests
	topicMap[addEncoding(p2p.RPCBlocksByRangeTopic)] = blockCollector

	return &limiter{limiterMap: topicMap, p2p: p2pProvider}
}

// Returns the current topic collector for the provided topic.
func (l *limiter) topicCollector(topic string) (*leakybucket.Collector, error) {
	l.RLock()
	defer l.RUnlock()

	collector, ok := l.limiterMap[topic]
	if !ok {
		return nil, errors.Errorf("collector does not exist for topic %s", topic)
	}
	return collector, nil
}

// validates a request with the accompanying cost.
func (l *limiter) validateRequest(stream network.Stream, amt uint64) error {
	l.RLock()
	defer l.RUnlock()

	topic := string(stream.Protocol())
	log := l.topicLogger(topic)

	collector, err := l.topicCollector(topic)
	if err != nil {
		return err
	}
	key := stream.Conn().RemotePeer().String()
	remaining := collector.Remaining(key)
	if amt > uint64(remaining) {
		l.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
		if l.p2p.Peers().IsBad(stream.Conn().RemotePeer()) {
			log.Debug("Disconnecting bad peer")
			defer func() {
				if err := l.p2p.Disconnect(stream.Conn().RemotePeer()); err != nil {
					log.WithError(err).Error("Failed to disconnect peer")
				}
			}()
		}
		writeErrorResponseToStream(responseCodeInvalidRequest, rateLimitedError, stream, l.p2p)
		return errors.New(rateLimitedError)
	}
	return nil
}

// adds the cost to our leaky bucket for the topic.
func (l *limiter) add(stream network.Stream, amt int64) {
	l.Lock()
	defer l.Unlock()

	topic := string(stream.Protocol())
	log := l.topicLogger(topic)

	collector, err := l.topicCollector(topic)
	if err != nil {
		log.Errorf("collector with topic '%s' does not exist", topic)
		return
	}
	key := stream.Conn().RemotePeer().String()
	collector.Add(key, amt)
}

// frees all the collectors and removes them.
func (l *limiter) free() {
	l.Lock()
	defer l.Unlock()

	tempMap := map[uintptr]bool{}
	for t, collector := range l.limiterMap {
		// Check if collector has already been cleared of
		// as all collectors are not distinct from each other.
		ptr := reflect.ValueOf(collector).Pointer()
		if tempMap[ptr] {
			continue
		}
		collector.Free()
		// Remove from map
		delete(l.limiterMap, t)
		tempMap[ptr] = true
	}
}

func (l *limiter) topicLogger(topic string) *logrus.Entry {
	return log.WithField("rate limiter", topic)
}
