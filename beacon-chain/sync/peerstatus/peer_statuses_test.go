package peerstatus

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

func TestLastUpdated(t *testing.T) {
	testID := peer.ID("test")
	status := &pb.Status{
		FinalizedEpoch: 1,
	}
	Set(testID, status)
	firstUpdated := LastUpdated(testID)

	time.Sleep(100 * time.Millisecond)

	status = &pb.Status{
		FinalizedEpoch: 2,
	}
	Set(testID, status)
	secondUpdated := LastUpdated(testID)

	if !secondUpdated.After(firstUpdated) {
		t.Error("lastupdated did not increment on subsequent set")
	}
}
