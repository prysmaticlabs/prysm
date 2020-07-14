package sync

import (
	"github.com/kevinms/leakybucket-go"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

const defaultBurstLimit = 5

type limiter struct {
	limiterMap map[string]*leakybucket.Collector
}

func NewRateLimiter(encoder p2p.EncodingProvider) *limiter {
	// add encoding suffix
	addEncoding := func(topic string) string {
		return topic + encoder.Encoding().ProtocolSuffix()
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

	return &limiter{limiterMap: topicMap}
}

func (l *limiter) topicCollector(topic string) (*leakybucket.Collector, error) {
	collector, ok := l.limiterMap[topic]
	if !ok {
		return nil, errors.Errorf("collector does not exist for topic %s", topic)
	}
	return collector, nil
}

func (l *limiter) free() {
	for t, collector := range l.limiterMap {
		collector.Free()
		// Remove from map
		delete(l.limiterMap, t)
	}
}
