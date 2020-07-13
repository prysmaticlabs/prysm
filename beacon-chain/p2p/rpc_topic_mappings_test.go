package p2p

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestVerifyRPCMappings(t *testing.T) {
	if err := VerifyTopicMapping(RPCStatusTopic, &pb.Status{}); err != nil {
		t.Errorf("Failed to verify status rpc topic: %v", err)
	}
	if err := VerifyTopicMapping(RPCStatusTopic, new([]byte)); err == nil {
		t.Errorf("Incorrect message type verified for status rpc topic")
	}

	if err := VerifyTopicMapping(RPCMetaDataTopic, new(interface{})); err != nil {
		t.Errorf("Failed to verify metadata rpc topic: %v", err)
	}
	if err := VerifyTopicMapping(RPCStatusTopic, new([]byte)); err == nil {
		t.Error("Incorrect message type verified for metadata rpc topic")
	}

	// TODO(#6408) Remove once issue is resolved
	if err := VerifyTopicMapping(RPCBlocksByRootTopic, [][32]byte{}); err != nil {
		t.Errorf("Failed to verify blocks by root rpc topic: %v", err)
	}
	if err := VerifyTopicMapping(RPCBlocksByRootTopic, new(pb.BeaconBlocksByRootRequest)); err != nil {
		t.Errorf("Failed to verify blocks by root rpc topic: %v", err)
	}
}
