package encoder

import (
	"io"
)

// Defines the different encoding formats
const (
	SSZ       = "ssz"        // SSZ is SSZ only.
	SSZSnappy = "ssz-snappy" // SSZSnappy is SSZ with snappy compression.
)

// NetworkEncoding represents an encoder compatible with Ethereum 2.0 p2p.
type NetworkEncoding interface {
	// Decodes to the provided message. The interface must be a pointer to the decoding destination.
	Decode([]byte, interface{}) error
	// DecodeWithLength a bytes from a reader with a varint length prefix. The interface must be a pointer to the
	// decoding destination.
	DecodeWithLength(io.Reader, interface{}) error
	// Encode an arbitrary message to the provided writer. The interface must be a pointer object to encode.
	Encode(io.Writer, interface{}) (int, error)
	// EncodeWithLength an arbitrary message to the provided writer with a varint length prefix. The interface must be
	// a pointer object to encode.
	EncodeWithLength(io.Writer, interface{}) (int, error)
	// ProtocolSuffix returns the last part of the protocol ID to indicate the encoding scheme.
	ProtocolSuffix() string
}
