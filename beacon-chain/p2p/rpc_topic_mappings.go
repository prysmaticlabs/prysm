package p2p

import (
	"reflect"

	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Current schema version for our rpc protocol ID.
const schemaVersionV1 = "/1"

const (
	// RPCStatusTopic defines the topic for the status rpc method.
	RPCStatusTopic = "/eth2/beacon_chain/req/status" + schemaVersionV1
	// RPCGoodByeTopic defines the topic for the goodbye rpc method.
	RPCGoodByeTopic = "/eth2/beacon_chain/req/goodbye" + schemaVersionV1
	// RPCBlocksByRangeTopic defines the topic for the blocks by range rpc method.
	RPCBlocksByRangeTopic = "/eth2/beacon_chain/req/beacon_blocks_by_range" + schemaVersionV1
	// RPCBlocksByRootTopic defines the topic for the blocks by root rpc method.
	RPCBlocksByRootTopic = "/eth2/beacon_chain/req/beacon_blocks_by_root" + schemaVersionV1
	// RPCPingTopic defines the topic for the ping rpc method.
	RPCPingTopic = "/eth2/beacon_chain/req/ping" + schemaVersionV1
	// RPCMetaDataTopic defines the topic for the metadata rpc method.
	RPCMetaDataTopic = "/eth2/beacon_chain/req/metadata" + schemaVersionV1
)

// RPCTopicMappings map the base message type to the rpc request.
var RPCTopicMappings = map[string]interface{}{
	RPCStatusTopic:        new(pb.Status),
	RPCGoodByeTopic:       new(uint64),
	RPCBlocksByRangeTopic: new(pb.BeaconBlocksByRangeRequest),
	RPCBlocksByRootTopic:  [][32]byte{},
	RPCPingTopic:          new(uint64),
	RPCMetaDataTopic:      new(interface{}),
}

// VerifyTopicMapping verifies that the topic and its accompanying
// message type is correct.
func VerifyTopicMapping(topic string, msg interface{}) error {
	msgType, ok := RPCTopicMappings[topic]
	if !ok {
		return errors.New("rpc topic is not registered currently")
	}
	receivedType := reflect.TypeOf(msg)
	registeredType := reflect.TypeOf(msgType)
	typeMatches := registeredType.AssignableTo(receivedType)

	// TODO(#6408) Allow multiple message types for topic, as we currently have 2 different
	// rpc request types until issue is resolved.
	if topic == RPCBlocksByRootTopic {
		if typeMatches {
			return nil
		}
		secondType := reflect.TypeOf(new(pb.BeaconBlocksByRootRequest))
		secondTypeMatches := secondType.AssignableTo(receivedType)
		if !secondTypeMatches {
			return errors.Errorf("accompanying message type is incorrect for topic: wanted %v or %v but got %v",
				registeredType.String(), secondType.String(), receivedType.String())
		}
		return nil
	}
	if !typeMatches {
		return errors.Errorf("accompanying message type is incorrect for topic: wanted %v  but got %v",
			registeredType.String(), receivedType.String())
	}
	return nil
}
