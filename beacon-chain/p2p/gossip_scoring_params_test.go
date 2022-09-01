package p2p

import (
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

func TestLoggingParameters(_ *testing.T) {
	logGossipParameters("testing", nil)
	logGossipParameters("testing", &pubsub.TopicScoreParams{})
	// Test out actual gossip parameters.
	logGossipParameters("testing", defaultBlockTopicParams())
	p := defaultAggregateSubnetTopicParams(10000)
	logGossipParameters("testing", p)
	p = defaultAggregateTopicParams(10000)
	logGossipParameters("testing", p)
	logGossipParameters("testing", defaultAttesterSlashingTopicParams())
	logGossipParameters("testing", defaultProposerSlashingTopicParams())
	logGossipParameters("testing", defaultVoluntaryExitTopicParams())
}
