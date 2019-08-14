package p2p

import (
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const (
	// overlay parameters
	gossipSubD   = 6  // topic stable mesh target count
	gossipSubDlo = 4  // topic stable mesh low watermark
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
	if pubsub.GossipSubD != gossipSubD {
		t.Errorf("gossipSubD, wanted: %d, got: %d", gossipSubD, pubsub.GossipSubD)
	}

	if pubsub.GossipSubDlo != gossipSubDlo {
		t.Errorf("gossipSubDlo, wanted: %d, got: %d", gossipSubDlo, pubsub.GossipSubDlo)
	}

	if pubsub.GossipSubDhi != gossipSubDhi {
		t.Errorf("gossipSubDhi, wanted: %d, got: %d", gossipSubDhi, pubsub.GossipSubDhi)
	}
}

func TestGossipParameters(t *testing.T) {
	if pubsub.GossipSubHistoryLength != gossipSubHistoryLength {
		t.Errorf("gossipSubHistoryLength, wanted: %d, got: %d", gossipSubHistoryLength, pubsub.GossipSubHistoryLength)
	}

	if pubsub.GossipSubHistoryGossip != gossipSubHistoryGossip {
		t.Errorf("gossipSubHistoryGossip, wanted: %d, got: %d", gossipSubHistoryGossip, pubsub.GossipSubDlo)
	}
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
	if pubsub.RandomSubD != randomSubD {
		t.Errorf("randomSubD, wanted: %d, got: %d", randomSubD, pubsub.RandomSubD)
	}
}
