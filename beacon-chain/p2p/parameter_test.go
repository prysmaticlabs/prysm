package p2p

import (
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestOverlayParameters(t *testing.T) {
	setPubSubParameters()
	assert.Equal(t, gossipSubD, pubsub.GossipSubD, "gossipSubD")
	assert.Equal(t, gossipSubDlo, pubsub.GossipSubDlo, "gossipSubDlo")
	assert.Equal(t, gossipSubDhi, pubsub.GossipSubDhi, "gossipSubDhi")
}

func TestGossipParameters(t *testing.T) {
	setPubSubParameters()
	assert.Equal(t, gossipSubMcacheLen, pubsub.GossipSubHistoryLength, "gossipSubMcacheLen")
	assert.Equal(t, gossipSubMcacheGossip, pubsub.GossipSubHistoryGossip, "gossipSubMcacheGossip")
	assert.Equal(t, gossipSubSeenTTL, int(pubsub.TimeCacheDuration.Milliseconds()/pubsub.GossipSubHeartbeatInterval.Milliseconds()), "gossipSubSeenTtl")
}

func TestFanoutParameters(t *testing.T) {
	setPubSubParameters()
	if pubsub.GossipSubFanoutTTL != gossipSubFanoutTTL {
		t.Errorf("gossipSubFanoutTTL, wanted: %d, got: %d", gossipSubFanoutTTL, pubsub.GossipSubFanoutTTL)
	}
}

func TestHeartbeatParameters(t *testing.T) {
	setPubSubParameters()
	if pubsub.GossipSubHeartbeatInterval != gossipSubHeartbeatInterval {
		t.Errorf("gossipSubHeartbeatInterval, wanted: %d, got: %d", gossipSubHeartbeatInterval, pubsub.GossipSubHeartbeatInterval)
	}
}

func TestMiscParameters(t *testing.T) {
	setPubSubParameters()
	assert.Equal(t, randomSubD, pubsub.RandomSubD, "randomSubD")
}
