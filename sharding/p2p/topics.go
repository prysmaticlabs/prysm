package p2p

import (
	"reflect"

	pb "github.com/ethereum/go-ethereum/sharding/p2p/proto"
)

// Mapping of message topic enums to protobuf types.
var topicTypeMapping = map[pb.Topic]reflect.Type{
	pb.Topic_COLLATION_BODY_REQUEST:  reflect.TypeOf(pb.CollationBodyRequest{}),
	pb.Topic_COLLATION_BODY_RESPONSE: reflect.TypeOf(pb.CollationBodyResponse{}),
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
