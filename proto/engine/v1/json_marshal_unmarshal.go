package enginev1

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
)

type executionPayloadAlias struct {
	*ExecutionPayload
}

func (e *executionPayloadAlias) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(e)
}

// MarshalJSON defines a custom json.Marshaler interface implementation
// that uses protojson underneath the hood, as protojson will respect
// proper struct tag naming conventions required for the JSON-RPC engine API to work.
func (e *ExecutionPayload) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		LogsBloom string `json:"logsBloom"`
		*executionPayloadAlias
	}{
		LogsBloom:             fmt.Sprintf("%#x", e.LogsBloom),
		executionPayloadAlias: (*executionPayloadAlias)(&executionPayloadAlias{ExecutionPayload: e}),
	})
}

// UnmarshalJSON defines a custom json.Unmarshaler interface implementation
// that uses protojson underneath the hood, as protojson will respect
// proper struct tag naming conventions required for the JSON-RPC engine API to work.
func (e *ExecutionPayload) UnmarshalJSON(enc []byte) error {
	return protojson.Unmarshal(enc, e)
}

// MarshalJSON --
func (p *PayloadAttributes) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(p)
}

// UnmarshalJSON --
func (p *PayloadAttributes) UnmarshalJSON(enc []byte) error {
	return protojson.Unmarshal(enc, p)
}

// MarshalJSON --
func (p *PayloadStatus) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(p)
}

// UnmarshalJSON --
func (p *PayloadStatus) UnmarshalJSON(enc []byte) error {
	return protojson.Unmarshal(enc, p)
}

// MarshalJSON --
func (f *ForkchoiceState) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(f)
}

// UnmarshalJSON --
func (f *ForkchoiceState) UnmarshalJSON(enc []byte) error {
	return protojson.Unmarshal(enc, f)
}
