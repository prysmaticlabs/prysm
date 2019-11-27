package kv

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
)


func decode(data []byte, dst proto.Message) error {
	enc, err := snappy.Decode(nil, data)
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(enc, dst); err != nil {
		return err
	}
	return nil
}

func encode(msg proto.Message) ([]byte, error) {
	enc, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return snappy.Encode(nil, enc), nil
}
