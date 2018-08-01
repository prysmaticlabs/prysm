package p2p

import (
	"reflect"

	beaconpb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	shardpb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
)

// Mapping of message topic enums to protobuf types.
var topicTypeMapping = map[pb.Topic]reflect.Type{
	pb.Topic_BEACON_BLOCK_HASH_ANNOUNCE:       reflect.TypeOf(pb.BeaconBlockHashAnnounce{}),
	pb.Topic_BEACON_BLOCK_REQUEST:             reflect.TypeOf(pb.BeaconBlockRequest{}),
	pb.Topic_BEACON_BLOCK_RESPONSE:            reflect.TypeOf(pb.BeaconBlockResponse{}),
	pb.Topic_COLLATION_BODY_REQUEST:           reflect.TypeOf(pb.CollationBodyRequest{}),
	pb.Topic_COLLATION_BODY_RESPONSE:          reflect.TypeOf(pb.CollationBodyResponse{}),
	pb.Topic_TRANSACTIONS:                     reflect.TypeOf(pb.Transaction{}),
	pb.Topic_CRYSTALLIZED_STATE_HASH_ANNOUNCE: reflect.TypeOf(pb.CrystallizedStateHashAnnounce{}),
	pb.Topic_CRYSTALLIZED_STATE_REQUEST:       reflect.TypeOf(pb.CrystallizedStateRequest{}),
	pb.Topic_CRYSTALLIZED_STATE_RESPONSE:      reflect.TypeOf(pb.CrystallizedStateResponse{}),
	pb.Topic_ACTIVE_STATE_HASH_ANNOUNCE:       reflect.TypeOf(pb.ActiveStateHashAnnounce{}),
	pb.Topic_ACTIVE_STATE_REQUEST:             reflect.TypeOf(pb.ActiveStateRequest{}),
	pb.Topic_ACTIVE_STATE_RESPONSE:            reflect.TypeOf(pb.ActiveStateResponse{}),
}

// Mapping of message types to topic enums.
var typeTopicMapping = reverseMapping(topicTypeMapping)

// ReverseMapping from K,V to V,K
func reverseMapping(m map[shardpb.Topic]reflect.Type) map[reflect.Type]shardpb.Topic {
	n := make(map[reflect.Type]shardpb.Topic)
	for k, v := range m {
		n[v] = k
	}
	return n
}

// These functions return the given topic for a given interface. This is the preferred
// way to resolve a topic from an value. The msg could be a pointer or value
// argument to resolve to the correct topic.
func topic(msg interface{}) shardpb.Topic {
	msgType := reflect.TypeOf(msg)
	if msgType.Kind() == reflect.Ptr {
		msgType = reflect.Indirect(reflect.ValueOf(msg)).Type()
	}
	return typeTopicMapping[msgType]
}
