package testutil

import (
	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/proto"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            true,
	ValidateJsonRawMessage: true,
	TagKey:                 "spec-name",
}.Froze()

// ConvertToPb converts some JSON compatible struct to given protobuf.
func ConvertToPb(i interface{}, p proto.Message) error {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, p)
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalYaml using a customized json encoder that supports "spec-name"
// override tag.
func UnmarshalYaml(y []byte, dest interface{}) error {
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, dest)
}
