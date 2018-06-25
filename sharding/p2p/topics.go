package p2p

import (
	"reflect"

	pb "github.com/ethereum/go-ethereum/sharding/p2p/proto"
)

// Mapping of message topic enums to protobuf types.
var topicTypeMapping = map[pb.Message_Topic]reflect.Type{
	pb.Message_COLLATION_BODY_REQUEST:  reflect.TypeOf(pb.CollationBodyRequest{}),
	pb.Message_COLLATION_BODY_RESPONSE: reflect.TypeOf(pb.CollationBodyResponse{}),
}

// Mapping of message types to topics.
var typeTopicMapping = reverseMapping(topicTypeMapping)

// Reverse map from K,V to V,K
func reverseMapping(m map[pb.Message_Topic]reflect.Type) map[reflect.Type]pb.Message_Topic {
	n := make(map[reflect.Type]pb.Message_Topic)
	for k, v := range m {
		n[v] = k
	}
	return n
}
