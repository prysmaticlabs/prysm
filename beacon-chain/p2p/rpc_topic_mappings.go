package p2p

import (
	"reflect"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	pb "github.com/prysmaticlabs/prysm/proto/proto/prysm/v2"
)

// SchemaVersionV1 specifies the schema version for our rpc protocol ID.
const SchemaVersionV1 = "/1"

// Specifies the protocol prefix for all our Req/Resp topics.
const protocolPrefix = "/eth2/beacon_chain/req"

// Specifies the name for the status message topic.
const statusMessageName = "/status"

// Specifies the name for the goodbye message topic.
const goodbyeMessageName = "/goodbye"

// Specifies the name for the beacon blocks by range message topic.
const beaconBlocksByRangeMessageName = "/beacon_blocks_by_range"

// Specifies the name for the beacon blocks by root message topic.
const beaconBlocksByRootsMessageName = "/beacon_blocks_by_root"

// Specifies the name for the ping message topic.
const pingMessageName = "/ping"

// Specifies the name for the metadata message topic.
const metadataMessageName = "/metadata"

const (
	// V1 RPC Topics
	// RPCStatusTopicV1 defines the v1 topic for the status rpc method.
	RPCStatusTopicV1 = protocolPrefix + statusMessageName + SchemaVersionV1
	// RPCGoodByeTopicV1 defines the v1 topic for the goodbye rpc method.
	RPCGoodByeTopicV1 = protocolPrefix + goodbyeMessageName + SchemaVersionV1
	// RPCBlocksByRangeTopicV1 defines v1 the topic for the blocks by range rpc method.
	RPCBlocksByRangeTopicV1 = protocolPrefix + beaconBlocksByRangeMessageName + SchemaVersionV1
	// RPCBlocksByRootTopicV1 defines the v1 topic for the blocks by root rpc method.
	RPCBlocksByRootTopicV1 = protocolPrefix + beaconBlocksByRootsMessageName + SchemaVersionV1
	// RPCPingTopicV1 defines the v1 topic for the ping rpc method.
	RPCPingTopicV1 = protocolPrefix + pingMessageName + SchemaVersionV1
	// RPCMetaDataTopicV1 defines the v1 topic for the metadata rpc method.
	RPCMetaDataTopicV1 = protocolPrefix + metadataMessageName + SchemaVersionV1
)

// RPCTopicMappings map the base message type to the rpc request.
var RPCTopicMappings = map[string]interface{}{
	RPCStatusTopicV1:        new(pb.Status),
	RPCGoodByeTopicV1:       new(types.SSZUint64),
	RPCBlocksByRangeTopicV1: new(pb.BeaconBlocksByRangeRequest),
	RPCBlocksByRootTopicV1:  new(p2ptypes.BeaconBlockByRootsReq),
	RPCPingTopicV1:          new(types.SSZUint64),
	RPCMetaDataTopicV1:      new(interface{}),
}

// Maps all registered protocol prefixes.
var protocolMapping = map[string]bool{
	protocolPrefix: true,
}

// Maps all the protocol message names for the different rpc
// topics.
var messageMapping = map[string]bool{
	statusMessageName:              true,
	goodbyeMessageName:             true,
	beaconBlocksByRangeMessageName: true,
	beaconBlocksByRootsMessageName: true,
	pingMessageName:                true,
	metadataMessageName:            true,
}

var versionMapping = map[string]bool{
	SchemaVersionV1: true,
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

	if !typeMatches {
		return errors.Errorf("accompanying message type is incorrect for topic: wanted %v  but got %v",
			registeredType.String(), receivedType.String())
	}
	return nil
}

// TopicDeconstructor splits the provided topic to its logical sub-sections.
// It is assumed all input topics will follow the specific schema:
// /protocol-prefix/message-name/schema-version/...
// For the purposes of deconstruction, only the first 3 components are
// relevant.
func TopicDeconstructor(topic string) (string, string, string, error) {
	origTopic := topic
	protPrefix := ""
	message := ""
	version := ""

	// Iterate through all the relevant mappings to find the relevant prefixes,messages
	// and version for this topic.
	for k := range protocolMapping {
		keyLen := len(k)
		if keyLen > len(topic) {
			continue
		}
		if topic[:keyLen] == k {
			protPrefix = k
			topic = topic[keyLen:]
		}
	}

	if protPrefix == "" {
		return "", "", "", errors.Errorf("unable to find a valid protocol prefix for %s", origTopic)
	}

	for k := range messageMapping {
		keyLen := len(k)
		if keyLen > len(topic) {
			continue
		}
		if topic[:keyLen] == k {
			message = k
			topic = topic[keyLen:]
		}
	}

	if message == "" {
		return "", "", "", errors.Errorf("unable to find a valid message for %s", origTopic)
	}

	for k := range versionMapping {
		keyLen := len(k)
		if keyLen > len(topic) {
			continue
		}
		if topic[:keyLen] == k {
			version = k
			topic = topic[keyLen:]
		}
	}

	if version == "" {
		return "", "", "", errors.Errorf("unable to find a valid schema version for %s", origTopic)
	}

	return protPrefix, message, version, nil
}

// RPCTopic is a type used to denote and represent a req/resp topic.
type RPCTopic string

// ProtocolPrefix returns the protocol prefix of the rpc topic.
func (r RPCTopic) ProtocolPrefix() string {
	prefix, _, _, err := TopicDeconstructor(string(r))
	if err != nil {
		return ""
	}
	return prefix
}

// MessageType returns the message type of the rpc topic.
func (r RPCTopic) MessageType() string {
	_, message, _, err := TopicDeconstructor(string(r))
	if err != nil {
		return ""
	}
	return message
}

// Version returns the schema version of the rpc topic.
func (r RPCTopic) Version() string {
	_, _, version, err := TopicDeconstructor(string(r))
	if err != nil {
		return ""
	}
	return version
}
