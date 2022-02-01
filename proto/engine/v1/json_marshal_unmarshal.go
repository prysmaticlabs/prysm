package enginev1

import "google.golang.org/protobuf/encoding/protojson"

// MarshalJSON defines a custom json.Marshaler interface implementation
// that uses protojson underneath the hood, as protojson will respect
// proper struct tag naming conventions required for the JSON-RPC engine API to work.
func (e *ExecutionPayload) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(e)
}

// UnmarshalJSON defines a custom json.Unmarshaler interface implementation
// that uses protojson underneath the hood, as protojson will respect
// proper struct tag naming conventions required for the JSON-RPC engine API to work.
func (e *ExecutionPayload) UnmarshalJSON(enc []byte) error {
	return protojson.Unmarshal(enc, e)
}

// MarshalJSON --
func (e *ExecutionPayloadHeader) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(e)
}

// UnmarshalJSON --
func (e *ExecutionPayloadHeader) UnmarshalJSON(enc []byte) error {
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
