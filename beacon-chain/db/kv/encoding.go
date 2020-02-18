package kv

import (
	"errors"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
)

func decode(data []byte, dst proto.Message) error {
	data, err := snappy.Decode(nil, data)
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(data, dst); err != nil {
		return err
	}
	return nil
}

func encode(msg proto.Message) ([]byte, error) {
	if msg == nil || reflect.ValueOf(msg).IsNil() {
		return nil, errors.New("cannot encode nil message")
	}
	enc, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return snappy.Encode(nil, enc), nil
}
