package spectest

import (
	"bytes"
	"encoding/json"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/jsonpb"
)

// TODO: Maybe this can go in testutils?

// Convert some JSON compatible struct to given protobuf.
func convertToPb(i interface{}, p proto.Message) error {
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