package p2p

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestVerifyRPCMappings(t *testing.T) {
	assert.NoError(t, VerifyTopicMapping(RPCStatusTopic, &pb.Status{}), "Failed to verify status rpc topic")
	assert.NotNil(t, VerifyTopicMapping(RPCStatusTopic, new([]byte)), "Incorrect message type verified for status rpc topic")

	assert.NoError(t, VerifyTopicMapping(RPCMetaDataTopic, new(interface{})), "Failed to verify metadata rpc topic")
	assert.NotNil(t, VerifyTopicMapping(RPCStatusTopic, new([]byte)), "Incorrect message type verified for metadata rpc topic")

	assert.NoError(t, VerifyTopicMapping(RPCBlocksByRootTopic, new(types.BeaconBlockByRootsReq)), "Failed to verify blocks by root rpc topic")
}
