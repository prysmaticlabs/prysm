package ssz

import (
	"github.com/gogo/protobuf/proto"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
)

var _ = encoder.NetworkEncoding(&SszNetworkEncoder{})

type SszNetworkEncoder struct {
	UseSnappyCompression bool
}

func (SszNetworkEncoder) Encode(msg proto.Message) ([]byte, error) {
	return nil, nil
}

func (SszNetworkEncoder) DecodeTo(b []byte, to proto.Message) error {
	return nil
}
