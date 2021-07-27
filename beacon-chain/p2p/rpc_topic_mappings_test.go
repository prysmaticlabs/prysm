package p2p

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestVerifyRPCMappings(t *testing.T) {
	assert.NoError(t, VerifyTopicMapping(RPCStatusTopicV1, &pb.Status{}), "Failed to verify status rpc topic")
	assert.NotNil(t, VerifyTopicMapping(RPCStatusTopicV1, new([]byte)), "Incorrect message type verified for status rpc topic")

	assert.NoError(t, VerifyTopicMapping(RPCMetaDataTopicV1, new(interface{})), "Failed to verify metadata rpc topic")
	assert.NotNil(t, VerifyTopicMapping(RPCStatusTopicV1, new([]byte)), "Incorrect message type verified for metadata rpc topic")

	assert.NoError(t, VerifyTopicMapping(RPCBlocksByRootTopicV1, new(types.BeaconBlockByRootsReq)), "Failed to verify blocks by root rpc topic")
}

func TestTopicDeconstructor(t *testing.T) {
	tt := []struct {
		name          string
		topic         string
		expectedError string
		output        []string
	}{
		{
			name:          "invalid topic",
			topic:         "/sjdksfks/dusidsdsd/ssz",
			expectedError: "unable to find a valid protocol prefix for /sjdksfks/dusidsdsd/ssz",
			output:        []string{"", "", ""},
		},
		{
			name:          "valid status topic",
			topic:         protocolPrefix + statusMessageName + SchemaVersionV1,
			expectedError: "",
			output:        []string{protocolPrefix, statusMessageName, SchemaVersionV1},
		},
		{
			name:          "malformed status topic",
			topic:         protocolPrefix + "/statis" + SchemaVersionV1,
			expectedError: "unable to find a valid message for /eth2/beacon_chain/req/statis/1",
			output:        []string{""},
		},
		{
			name:          "valid beacon block by range topic",
			topic:         protocolPrefix + beaconBlocksByRangeMessageName + SchemaVersionV1 + "/ssz_snappy",
			expectedError: "",
			output:        []string{protocolPrefix, beaconBlocksByRangeMessageName, SchemaVersionV1},
		},
		{
			name:          "beacon block by range topic with malformed version",
			topic:         protocolPrefix + beaconBlocksByRangeMessageName + "/v" + "/ssz_snappy",
			expectedError: "unable to find a valid schema version for /eth2/beacon_chain/req/beacon_blocks_by_range/v/ssz_snappy",
			output:        []string{""},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			protocolPref, message, version, err := TopicDeconstructor(test.topic)
			if test.expectedError != "" {
				require.NotNil(t, err)
				assert.Equal(t, test.expectedError, err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.output[0], protocolPref)
				assert.Equal(t, test.output[1], message)
				assert.Equal(t, test.output[2], version)
			}
		})
	}
}
