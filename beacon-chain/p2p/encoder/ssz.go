package encoder

import (
	"io"

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

func (e SszNetworkEncoder) doEncode(msg interface{}) ([]byte, error) {
	b, err := ssz.Marshal(msg)
	if err != nil {
		return nil, err
	}
	if e.UseSnappyCompression {
		b = snappy.Encode(nil /*dst*/, b)
	}
	return b, nil
}

// Encode the proto message to the io.Writer.
func (e SszNetworkEncoder) Encode(w io.Writer, msg interface{}) (int, error) {
	if msg == nil {
		return 0, nil
	}

	b, err := e.doEncode(msg)
	if err != nil {
		return 0, err
	}
	return w.Write(b)
}

// EncodeWithLength the proto message to the io.Writer. This encoding prefixes the byte slice with a protobuf varint
// to indicate the size of the message.
func (e SszNetworkEncoder) EncodeWithLength(w io.Writer, msg interface{}) (int, error) {
	if msg == nil {
		return 0, nil
	}
	b, err := e.doEncode(msg)
	if err != nil {
		return 0, err
	}
	b = append(proto.EncodeVarint(uint64(len(b))), b...)
	return w.Write(b)
}

// Decode the bytes to the protobuf message provided.
func (e SszNetworkEncoder) Decode(b []byte, to interface{}) error {
	if e.UseSnappyCompression {
		var err error
		b, err = snappy.Decode(nil /*dst*/, b)
		if err != nil {
			return err
		}
	}

	return ssz.Unmarshal(b, to)
}

// DecodeWithLength the bytes from io.Reader to the protobuf message provided.
func (e SszNetworkEncoder) DecodeWithLength(r io.Reader, to interface{}) error {
	msgLen, err := readVarint(r)
	if err != nil {
		return err
	}
	b := make([]byte, msgLen)
	_, err = r.Read(b)
	if err != nil {
		return err
	}
	if e.UseSnappyCompression {
		var err error
		b, err = snappy.Decode(nil /*dst*/, b)
		if err != nil {
			return err
		}
	}
	return ssz.Unmarshal(b, to)
}

// ProtocolSuffix returns the appropriate suffix for protocol IDs.
func (e SszNetworkEncoder) ProtocolSuffix() string {
	if e.UseSnappyCompression {
		return "/ssz_snappy"
	}
	return "/ssz"
}
