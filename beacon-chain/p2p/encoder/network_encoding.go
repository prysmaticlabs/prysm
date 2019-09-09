package encoder

import (
	"io"

	"github.com/gogo/protobuf/proto"
)

// Defines the different encoding formats
const (
	SSZ       = "ssz"        // SSZ is SSZ only.
	SSZSnappy = "ssz-snappy" // SSZSnappy is SSZ with snappy compression.
)

// NetworkEncoding represents an encoder compatible with Ethereum 2.0 p2p.
type NetworkEncoding interface {
	// Decodes to the provided message.
	Decode([]byte, proto.Message) error
	// DecodeWithLength a bytes from a reader with a varint length prefix.
	DecodeWithLength(io.Reader, proto.Message) error
	// Encode an arbitrary message to the provided writer.
	Encode(io.Writer, proto.Message) (int, error)
	// EncodeWithLength an arbitrary message to the provided writer with a varint length prefix.
	EncodeWithLength(io.Writer, proto.Message) (int, error)
	// ProtocolSuffix returns the last part of the protocol ID to indicate the encoding scheme.
	ProtocolSuffix() string
}
