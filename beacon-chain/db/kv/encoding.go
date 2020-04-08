package kv

import (
	"errors"
	"reflect"

	fastssz "github.com/ferranbt/fastssz"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
)

func decode(data []byte, dst proto.Message) error {
	data, err := snappy.Decode(nil, data)
	if err != nil {
		return err
	}
	if mm, ok := dst.(fastssz.Unmarshaler); ok {
		return mm.UnmarshalSSZ(data)
	}
	return proto.Unmarshal(data, dst)
}

func encode(msg proto.Message) ([]byte, error) {
	if msg == nil || reflect.ValueOf(msg).IsNil() {
		return nil, errors.New("cannot encode nil message")
	}
	var enc []byte
	var err error
	if mm, ok := msg.(fastssz.Marshaler); ok {
		enc, err = mm.MarshalSSZ()
		if err != nil {
			return nil, err
		}
	} else {
		enc, err = proto.Marshal(msg)
		if err != nil {
			return nil, err
		}
	}
	return snappy.Encode(nil, enc), nil
}
