package peerstatus

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
)

func TestIncrementFailureCount(t *testing.T) {
	testID := peer.ID("test")
	IncreaseFailureCount(testID)
	if FailureCount(testID) != 1 {
		t.Errorf("Wanted failure count of %d but got %d", 1, FailureCount(testID))
	}
}

func TestAboveFailureThreshold(t *testing.T) {
	testID := peer.ID("test")
	for i := 0; i <= maxFailureThreshold; i++ {
		IncreaseFailureCount(testID)
	}
	if !IsBadPeer(testID) {
		t.Errorf("Peer isnt considered as a bad peer despite crossing the failure threshold "+
			"with a failure count of %d", FailureCount(testID))
	}
}
