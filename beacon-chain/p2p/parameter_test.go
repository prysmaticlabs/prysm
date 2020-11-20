package p2p

import (
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

const (
	// overlay parameters
	gossipSubD   = 8  // topic stable mesh target count
	gossipSubDlo = 6  // topic stable mesh low watermark
	gossipSubDhi = 12 // topic stable mesh high watermark

	// gossip parameters
	gossipSubMcacheLen    = 6   // number of windows to retain full messages in cache for `IWANT` responses
	gossipSubMcacheGossip = 3   // number of windows to gossip about
	gossipSubSeenTTL      = 550 // number of heartbeat intervals to retain message IDs

	// fanout ttl
	gossipSubFanoutTTL = 60000000000 // TTL for fanout maps for topics we are not subscribed to but have published to, in nano seconds

	// heartbeat interval
	gossipSubHeartbeatInterval = 700 * time.Millisecond // frequency of heartbeat, milliseconds

	// misc
	randomSubD = 6 // random gossip target
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
