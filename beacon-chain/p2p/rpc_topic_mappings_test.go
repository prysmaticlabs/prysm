package p2p

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestVerifyRPCMappings(t *testing.T) {
	assert.NoError(t, VerifyTopicMapping(RPCStatusTopic, &pb.Status{}), "Failed to verify status rpc topic")
	assert.NotNil(t, VerifyTopicMapping(RPCStatusTopic, new([]byte)), "Incorrect message type verified for status rpc topic")

	assert.NoError(t, VerifyTopicMapping(RPCMetaDataTopic, new(interface{})), "Failed to verify metadata rpc topic")
	assert.NotNil(t, VerifyTopicMapping(RPCStatusTopic, new([]byte)), "Incorrect message type verified for metadata rpc topic")

	// TODO(#6408) Remove once issue is resolved
	assert.NoError(t, VerifyTopicMapping(RPCBlocksByRootTopic, [][32]byte{}), "Failed to verify blocks by root rpc topic")
	assert.NoError(t, VerifyTopicMapping(RPCBlocksByRootTopic, new(pb.BeaconBlocksByRootRequest)), "Failed to verify blocks by root rpc topic")
}
