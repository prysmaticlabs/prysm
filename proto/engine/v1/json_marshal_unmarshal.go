package enginev1

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

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
		LogsBloom []byte `json:"logsBloom"`
		*executionPayloadAlias
	}{
		LogsBloom: hexBytes(e.LogsBloom),
		executionPayloadAlias: (*executionPayloadAlias)(&executionPayloadAlias{
			ExecutionPayload: e,
		}),
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

type hexBytes []byte
type quantity uint64

func (b hexBytes) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%#x", b)), nil
}

func (b hexBytes) UnmarshalJSON(enc []byte) error {
	decoded, err := hex.DecodeString(strings.TrimPrefix(string(enc), "0x"))
	if err != nil {
		return err
	}
	b = decoded
	return nil
}

func (q quantity) MarshalJSON() ([]byte, error) {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, uint64(q))
	return enc, nil
}

func (q quantity) UnmarshalJSON(enc []byte) error {
	decoded, err := hex.DecodeString(strings.TrimPrefix(string(enc), "0x"))
	if err != nil {
		return err
	}
	q = quantity(binary.BigEndian.Uint64(decoded))
	return nil
}
