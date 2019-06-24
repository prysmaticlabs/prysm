package testutil

import (
	"bytes"
	"encoding/json"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

// ConvertToPb converts some JSON compatible struct to given protobuf.
func ConvertToPb(i interface{}, p proto.Message) error {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}
	err = jsonpb.Unmarshal(bytes.NewReader(b), p)
	if err != nil {
		return err
	}
	return nil
}
