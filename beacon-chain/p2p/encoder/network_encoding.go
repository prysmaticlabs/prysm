package encoder

import "github.com/gogo/protobuf/proto"

// NetworkEncoding represents an encoder compatible with Ethereum 2.0 p2p.
type NetworkEncoding interface {
	// DecodeTo accepts a byte slice and decodes it to the provided message.
	DecodeTo([]byte, proto.Message) error
	// Encode an arbitrary message to bytes.
	Encode(proto.Message) ([]byte, error)
}
