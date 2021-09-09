package sync

import (
	"reflect"
	"sync"

	"github.com/kevinms/leakybucket-go"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/sirupsen/logrus"
	"github.com/trailofbits/go-mutexasserts"
)

const defaultBurstLimit = 5

// Dummy topic to validate all incoming rpc requests.
const rpcLimiterTopic = "rpc-limiter-topic"

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
	topicMap[addEncoding(p2p.RPCGoodByeTopicV1)] = leakybucket.NewCollector(1, 1, false /* deleteEmptyBuckets */)
	// MetadataV0 Message
	topicMap[addEncoding(p2p.RPCMetaDataTopicV1)] = leakybucket.NewCollector(1, defaultBurstLimit, false /* deleteEmptyBuckets */)
	topicMap[addEncoding(p2p.RPCMetaDataTopicV2)] = leakybucket.NewCollector(1, defaultBurstLimit, false /* deleteEmptyBuckets */)
	// Ping Message
	topicMap[addEncoding(p2p.RPCPingTopicV1)] = leakybucket.NewCollector(1, defaultBurstLimit, false /* deleteEmptyBuckets */)
	// Status Message
	topicMap[addEncoding(p2p.RPCStatusTopicV1)] = leakybucket.NewCollector(1, defaultBurstLimit, false /* deleteEmptyBuckets */)

	// Use a single collector for block requests
	blockCollector := leakybucket.NewCollector(allowedBlocksPerSecond, allowedBlocksBurst, false /* deleteEmptyBuckets */)
	// Collector for V2
	blockCollectorV2 := leakybucket.NewCollector(allowedBlocksPerSecond, allowedBlocksBurst, false /* deleteEmptyBuckets */)

	// BlocksByRoots requests
	topicMap[addEncoding(p2p.RPCBlocksByRootTopicV1)] = blockCollector
	topicMap[addEncoding(p2p.RPCBlocksByRootTopicV2)] = blockCollectorV2

	// BlockByRange requests
	topicMap[addEncoding(p2p.RPCBlocksByRangeTopicV1)] = blockCollector
	topicMap[addEncoding(p2p.RPCBlocksByRangeTopicV2)] = blockCollectorV2

	// General topic for all rpc requests.
	topicMap[rpcLimiterTopic] = leakybucket.NewCollector(5, defaultBurstLimit*2, false /* deleteEmptyBuckets */)

	return &limiter{limiterMap: topicMap, p2p: p2pProvider}
}

// Returns the current topic collector for the provided topic.
func (l *limiter) topicCollector(topic string) (*leakybucket.Collector, error) {
	l.RLock()
	defer l.RUnlock()
	return l.retrieveCollector(topic)
}

// validates a request with the accompanying cost.
func (l *limiter) validateRequest(stream network.Stream, amt uint64) error {
	l.RLock()
	defer l.RUnlock()

	topic := string(stream.Protocol())

	collector, err := l.retrieveCollector(topic)
	if err != nil {
		return err
	}
	key := stream.Conn().RemotePeer().String()
	remaining := collector.Remaining(key)
	// Treat each request as a minimum of 1.
	if amt == 0 {
		amt = 1
	}
	if amt > uint64(remaining) {
		l.p2p.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		writeErrorResponseToStream(responseCodeInvalidRequest, p2ptypes.ErrRateLimited.Error(), stream, l.p2p)
		return p2ptypes.ErrRateLimited
	}
	return nil
}

// This is used to validate all incoming rpc streams from external peers.
func (l *limiter) validateRawRpcRequest(stream network.Stream) error {
	l.RLock()
	defer l.RUnlock()

	topic := rpcLimiterTopic

	collector, err := l.retrieveCollector(topic)
	if err != nil {
		return err
	}
	key := stream.Conn().RemotePeer().String()
	remaining := collector.Remaining(key)
	// Treat each request as a minimum of 1.
	amt := int64(1)
	if amt > remaining {
		l.p2p.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		writeErrorResponseToStream(responseCodeInvalidRequest, p2ptypes.ErrRateLimited.Error(), stream, l.p2p)
		return p2ptypes.ErrRateLimited
	}
	return nil
}

// adds the cost to our leaky bucket for the topic.
func (l *limiter) add(stream network.Stream, amt int64) {
	l.Lock()
	defer l.Unlock()

	topic := string(stream.Protocol())
	log := l.topicLogger(topic)

	collector, err := l.retrieveCollector(topic)
	if err != nil {
		log.Errorf("collector with topic '%s' does not exist", topic)
		return
	}
	key := stream.Conn().RemotePeer().String()
	collector.Add(key, amt)
}

// adds the cost to our leaky bucket for the peer.
func (l *limiter) addRawStream(stream network.Stream) {
	l.Lock()
	defer l.Unlock()

	topic := rpcLimiterTopic
	log := l.topicLogger(topic)

	collector, err := l.retrieveCollector(topic)
	if err != nil {
		log.Errorf("collector with topic '%s' does not exist", topic)
		return
	}
	key := stream.Conn().RemotePeer().String()
	collector.Add(key, 1)
}

// frees all the collectors and removes them.
func (l *limiter) free() {
	l.Lock()
	defer l.Unlock()

	tempMap := map[uintptr]bool{}
	for t, collector := range l.limiterMap {
		// Check if collector has already been cleared off
		// as all collectors are not distinct from each other.
		ptr := reflect.ValueOf(collector).Pointer()
		if tempMap[ptr] {
			// Remove from map
			delete(l.limiterMap, t)
			continue
		}
		collector.Free()
		// Remove from map
		delete(l.limiterMap, t)
		tempMap[ptr] = true
	}
}

// not to be used outside the rate limiter file as it is unsafe for concurrent usage
// and is protected by a lock on all of its usages here.
func (l *limiter) retrieveCollector(topic string) (*leakybucket.Collector, error) {
	if !mutexasserts.RWMutexLocked(&l.RWMutex) && !mutexasserts.RWMutexRLocked(&l.RWMutex) {
		return nil, errors.New("limiter.retrieveCollector: caller must hold read/write lock")
	}
	collector, ok := l.limiterMap[topic]
	if !ok {
		return nil, errors.Errorf("collector does not exist for topic %s", topic)
	}
	return collector, nil
}

func (l *limiter) topicLogger(topic string) *logrus.Entry {
	return log.WithField("rate limiter", topic)
}
