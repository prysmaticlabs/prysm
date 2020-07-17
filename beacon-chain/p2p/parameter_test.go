package p2p

import (
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

const (
	// overlay parameters
	gossipSubD   = 6  // topic stable mesh target count
	gossipSubDlo = 5  // topic stable mesh low watermark
	gossipSubDhi = 12 // topic stable mesh high watermark

	// gossip parameters
	gossipSubHistoryLength = 5 // number of heartbeat intervals to retain message IDs
	gossipSubHistoryGossip = 3 // number of windows to gossip about

	// fanout ttl
	gossipSubFanoutTTL = 60000000000 // TTL for fanout maps for topics we are not subscribed to but have published to, in nano seconds

	// heartbeat interval
	gossipSubHeartbeatInterval = 1000000000 // frequency of heartbeat, in nano seconds

	// misc
	randomSubD = 6 // random gossip target
)

func TestOverlayParameters(t *testing.T) {
	assert.Equal(t, gossipSubD, pubsub.GossipSubD, "gossipSubD")
	assert.Equal(t, gossipSubDlo, pubsub.GossipSubDlo, "gossipSubDlo")
	assert.Equal(t, gossipSubDhi, pubsub.GossipSubDhi, "gossipSubDhi")
}

func TestGossipParameters(t *testing.T) {
	assert.Equal(t, gossipSubHistoryLength, pubsub.GossipSubHistoryLength, "gossipSubHistoryLength")
	assert.Equal(t, gossipSubHistoryGossip, pubsub.GossipSubHistoryGossip, "gossipSubHistoryGossip")
}

func TestFanoutParameters(t *testing.T) {
	if pubsub.GossipSubFanoutTTL != gossipSubFanoutTTL {
		t.Errorf("gossipSubFanoutTTL, wanted: %d, got: %d", gossipSubFanoutTTL, pubsub.GossipSubFanoutTTL)
	}
}

func TestHeartbeatParameters(t *testing.T) {
	if pubsub.GossipSubHeartbeatInterval != gossipSubHeartbeatInterval {
		t.Errorf("gossipSubHeartbeatInterval, wanted: %d, got: %d", gossipSubHeartbeatInterval, pubsub.GossipSubHeartbeatInterval)
	}
}

func TestMiscParameters(t *testing.T) {
	assert.Equal(t, randomSubD, pubsub.RandomSubD, "randomSubD")
}
