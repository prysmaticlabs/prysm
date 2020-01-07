package kv

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func decode(data []byte, dst proto.Message) error {
	if featureconfig.Get().EnableSnappyDBCompression {
		var err error
		data, err = snappy.Decode(nil, data)
		if err != nil {
			return err
		}
	}
	if err := proto.Unmarshal(data, dst); err != nil {
		return err
	}
	return nil
}

func encode(msg proto.Message) ([]byte, error) {
	enc, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	if !featureconfig.Get().EnableSnappyDBCompression {
		return enc, nil
	}

	return snappy.Encode(nil, enc), nil
}
