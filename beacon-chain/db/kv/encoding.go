package kv

import (
	"errors"
	"reflect"

	fastssz "github.com/ferranbt/fastssz"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func decode(data []byte, dst proto.Message) error {
	data, err := snappy.Decode(nil, data)
	if err != nil {
		return err
	}
	if mm, ok := dst.(fastssz.Unmarshaler); ok {
		return mm.UnmarshalSSZ(data)
	}
	switch dst.(type) {
	case *pb.BeaconState:
	case fastssz.Unmarshaler:
		return dst.(fastssz.Unmarshaler).UnmarshalSSZ(data)
	}
	return proto.Unmarshal(data, dst)
}

func encode(msg proto.Message) ([]byte, error) {
	if msg == nil || reflect.ValueOf(msg).IsNil() {
		return nil, errors.New("cannot encode nil message")
	}
	var enc []byte
	var err error
	switch msg.(type) {
	case *pb.BeaconState:
	case fastssz.Marshaler:
		enc, err = msg.(fastssz.Marshaler).MarshalSSZ()
		if err != nil {
			return nil, err
		}
	default:
		enc, err = proto.Marshal(msg)
		if err != nil {
			return nil, err
		}
	}
	return snappy.Encode(nil, enc), nil
}
