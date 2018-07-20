package p2p

import (
	"reflect"

	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
)

// Mapping of message topic enums to protobuf types.
var topicTypeMapping = map[pb.Topic]reflect.Type{
	pb.Topic_COLLATION_BODY_REQUEST:  reflect.TypeOf(pb.CollationBodyRequest{}),
	pb.Topic_COLLATION_BODY_RESPONSE: reflect.TypeOf(pb.CollationBodyResponse{}),
	pb.Topic_TRANSACTIONS:            reflect.TypeOf(pb.Transaction{}),
}

// Mapping of message types to topic enums.
var typeTopicMapping = reverseMapping(topicTypeMapping)

// ReverseMapping from K,V to V,K
func reverseMapping(m map[pb.Topic]reflect.Type) map[reflect.Type]pb.Topic {
	n := make(map[reflect.Type]pb.Topic)
	for k, v := range m {
		n[v] = k
	}
	return n
}

// Topic returns the given topic for a given interface. This is the preferred
// way to resolve a topic from an value. The msg could be a pointer or value
// argument to resolve to the correct topic.
func topic(msg interface{}) pb.Topic {
	msgType := reflect.TypeOf(msg)
	if msgType.Kind() == reflect.Ptr {
		msgType = reflect.Indirect(reflect.ValueOf(msg)).Type()
	}
	return typeTopicMapping[msgType]
}
