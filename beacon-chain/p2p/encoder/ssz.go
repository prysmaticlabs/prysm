package encoder

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prysmaticlabs/go-ssz"
)

var _ = NetworkEncoding(&SszNetworkEncoder{})

// SszNetworkEncoder supports p2p networking encoding using SimpleSerialize
// with snappy compression (if enabled).
type SszNetworkEncoder struct {
	UseSnappyCompression bool
}

// Encode the proto message to bytes.
func (e SszNetworkEncoder) Encode(msg proto.Message) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}

	b, err := ssz.Marshal(msg)
	if err != nil {
		return nil, err
	}
	if e.UseSnappyCompression {
		b = snappy.Encode(nil /*dst*/, b)
	}
	return b, nil
}

// DecodeTo decodes the bytes to the protobuf message provided.
func (e SszNetworkEncoder) DecodeTo(b []byte, to proto.Message) error {
	if e.UseSnappyCompression {
		var err error
		b, err = snappy.Decode(nil /*dst*/, b)
		if err != nil {
			return err
		}
	}

	return ssz.Unmarshal(b, to)
}
