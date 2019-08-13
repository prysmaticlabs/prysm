package ssz

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	gossz "github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
)

var _ = encoder.NetworkEncoding(&SszNetworkEncoder{})

type SszNetworkEncoder struct {
	UseSnappyCompression bool
}

func (e SszNetworkEncoder) Encode(msg proto.Message) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}

	b, err := gossz.Marshal(msg)
	if err != nil {
		return nil, err
	}
	if e.UseSnappyCompression {
		b = snappy.Encode(nil /*dst*/, b)
	}
	return b, nil
}

func (e SszNetworkEncoder) DecodeTo(b []byte, to proto.Message) error {
	if e.UseSnappyCompression {
		var err error
		b, err = snappy.Decode(nil /*dst*/, b)
		if err != nil {
			return err
		}
	}

	return gossz.Unmarshal(b, to)
}
