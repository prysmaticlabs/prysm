package p2p

import (
	"reflect"

	beaconpb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	shardpb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
)

// Mapping of message topic enums to protobuf types.
var topicTypeMapping = map[shardpb.Topic]reflect.Type{
	shardpb.Topic_COLLATION_BODY_REQUEST:     reflect.TypeOf(shardpb.CollationBodyRequest{}),
	shardpb.Topic_COLLATION_BODY_RESPONSE:    reflect.TypeOf(shardpb.CollationBodyResponse{}),
	shardpb.Topic_TRANSACTIONS:               reflect.TypeOf(shardpb.Transaction{}),
	shardpb.Topic_BEACON_BLOCK_HASH_ANNOUNCE: reflect.TypeOf(beaconpb.BeaconBlockHashAnnounce{}),
	shardpb.Topic_BEACON_BLOCK_REQUEST:       reflect.TypeOf(beaconpb.BeaconBlockRequest{}),
	shardpb.Topic_BEACON_BLOCK_RESPONSE:      reflect.TypeOf(beaconpb.BeaconBlockResponse{}),
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
