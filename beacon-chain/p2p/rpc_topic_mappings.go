package p2p

import (
	"reflect"

	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// RPCTopicMappings represent the protocol ID to protobuf message type map for easy
// lookup. These mappings should be used for outbound sending only. Peers may respond
// with a different message type as defined by the p2p protocol.
var RPCTopicMappings = map[string]interface{}{
	"/eth2/beacon_chain/req/hello/1":                &p2ppb.Hello{},
	"/eth2/beacon_chain/req/goodbye/1":              &p2ppb.Goodbye{},
	"/eth2/beacon_chain/req/beacon_blocks/1":        &p2ppb.BeaconBlocksRequest{},
	"/eth2/beacon_chain/req/recent_beacon_blocks/1": [][32]byte{},
}

// RPCTypeMapping is the inverse of RPCTopicMappings so that an arbitrary protobuf message
// can be mapped to a protocol ID string.
var RPCTypeMapping = make(map[reflect.Type]string)

func init() {
	for k, v := range RPCTopicMappings {
		RPCTypeMapping[reflect.TypeOf(v)] = k
	}
}
