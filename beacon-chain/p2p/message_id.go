package p2p

import (
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/network/forks"
)

// MsgID is a content addressable ID function.
//
// Ethereum Beacon Chain spec defines the message ID as:
//    The `message-id` of a gossipsub message MUST be the following 20 byte value computed from the message data:
//    If `message.data` has a valid snappy decompression, set `message-id` to the first 20 bytes of the `SHA256` hash of
//    the concatenation of `MESSAGE_DOMAIN_VALID_SNAPPY` with the snappy decompressed message data,
//    i.e. `SHA256(MESSAGE_DOMAIN_VALID_SNAPPY + snappy_decompress(message.data))[:20]`.
//
//    Otherwise, set `message-id` to the first 20 bytes of the `SHA256` hash of
//    the concatenation of `MESSAGE_DOMAIN_INVALID_SNAPPY` with the raw message data,
//    i.e. `SHA256(MESSAGE_DOMAIN_INVALID_SNAPPY + message.data)[:20]`.
func MsgID(genesisValidatorsRoot []byte, pmsg *pubsub_pb.Message) string {
	if pmsg == nil || pmsg.Data == nil || pmsg.Topic == nil {
		// Impossible condition that should
		// never be hit.
		msg := make([]byte, 20)
		copy(msg, "invalid")
		return string(msg)
	}
	digest, err := ExtractGossipDigest(*pmsg.Topic)
	if err != nil {
		// Impossible condition that should
		// never be hit.
		msg := make([]byte, 20)
		copy(msg, "invalid")
		return string(msg)
	}
	_, fEpoch, err := forks.RetrieveForkDataFromDigest(digest, genesisValidatorsRoot)
	if err != nil {
		// Impossible condition that should
		// never be hit.
		msg := make([]byte, 20)
		copy(msg, "invalid")
		return string(msg)
	}
	if fEpoch >= params.BeaconConfig().AltairForkEpoch {
		return postAltairMsgID(pmsg, fEpoch)
	}
	decodedData, err := encoder.DecodeSnappy(pmsg.Data, params.BeaconNetworkConfig().GossipMaxSize)
	if err != nil {
		combinedData := append(params.BeaconNetworkConfig().MessageDomainInvalidSnappy[:], pmsg.Data...)
		h := hash.Hash(combinedData)
		return string(h[:20])
	}
	combinedData := append(params.BeaconNetworkConfig().MessageDomainValidSnappy[:], decodedData...)
	h := hash.Hash(combinedData)
	return string(h[:20])
}

// Spec:
// The derivation of the message-id has changed starting with Altair to incorporate the message topic along with the message data.
// These are fields of the Message Protobuf, and interpreted as empty byte strings if missing. The message-id MUST be the following
// 20 byte value computed from the message:
//
// If message.data has a valid snappy decompression, set message-id to the first 20 bytes of the SHA256 hash of the concatenation of
// the following data: MESSAGE_DOMAIN_VALID_SNAPPY, the length of the topic byte string (encoded as little-endian uint64), the topic
// byte string, and the snappy decompressed message data: i.e. SHA256(MESSAGE_DOMAIN_VALID_SNAPPY + uint_to_bytes(uint64(len(message.topic)))
// + message.topic + snappy_decompress(message.data))[:20]. Otherwise, set message-id to the first 20 bytes of the SHA256 hash of the concatenation
// of the following data: MESSAGE_DOMAIN_INVALID_SNAPPY, the length of the topic byte string (encoded as little-endian uint64),
// the topic byte string, and the raw message data: i.e. SHA256(MESSAGE_DOMAIN_INVALID_SNAPPY + uint_to_bytes(uint64(len(message.topic))) + message.topic + message.data)[:20].
func postAltairMsgID(pmsg *pubsub_pb.Message, fEpoch types.Epoch) string {
	topic := *pmsg.Topic
	topicLen := uint64(len(topic))
	topicLenBytes := bytesutil.Uint64ToBytesLittleEndian(topicLen)

	// beyond Bellatrix epoch, allow 10 Mib gossip data size
	gossipPubSubSize := params.BeaconNetworkConfig().GossipMaxSize
	if fEpoch >= params.BeaconConfig().BellatrixForkEpoch {
		gossipPubSubSize = params.BeaconNetworkConfig().GossipMaxSizeBellatrix
	}

	decodedData, err := encoder.DecodeSnappy(pmsg.Data, gossipPubSubSize)
	if err != nil {
		totalLength := len(params.BeaconNetworkConfig().MessageDomainInvalidSnappy) + len(topicLenBytes) + int(topicLen) + len(pmsg.Data) // lint:ignore uintcast -- Message length cannot exceed int64
		combinedData := make([]byte, 0, totalLength)
		combinedData = append(combinedData, params.BeaconNetworkConfig().MessageDomainInvalidSnappy[:]...)
		combinedData = append(combinedData, topicLenBytes...)
		combinedData = append(combinedData, topic...)
		combinedData = append(combinedData, pmsg.Data...)
		h := hash.Hash(combinedData)
		return string(h[:20])
	}
	totalLength := len(params.BeaconNetworkConfig().MessageDomainValidSnappy) + len(topicLenBytes) + int(topicLen) + len(decodedData) // lint:ignore uintcast -- Message length cannot exceed int64
	combinedData := make([]byte, 0, totalLength)
	combinedData = append(combinedData, params.BeaconNetworkConfig().MessageDomainValidSnappy[:]...)
	combinedData = append(combinedData, topicLenBytes...)
	combinedData = append(combinedData, topic...)
	combinedData = append(combinedData, decodedData...)
	h := hash.Hash(combinedData)
	return string(h[:20])
}
