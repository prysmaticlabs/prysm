package v1

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDecodeOpts(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		res    interface{}
		err    error
		result string
	}{
		{
			name:  "EmptyInput",
			input: "",
			res: &struct {
				Name string `json:"name,omitempty"`
			}{},
			result: `{}`,
		},
		{
			name:  "EmptyJSON",
			input: "{}",
			res: &struct {
				Name string `json:"name,omitempty"`
			}{},
			result: `{}`,
		},
		{
			name:  "BadInput",
			input: "bad",
			res: &struct {
				Name string `json:"name,omitempty"`
			}{},
			err: errors.New("open bad: no such file or directory"),
		},
		{
			name:  "GoodDirect",
			input: `{"name":"test"}`,
			res: &struct {
				Name string `json:"name,omitempty"`
			}{},
			result: `{"name":"test"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := decodeOpts(test.input, test.res)
			if test.err == nil {
				require.NoError(t, err)
				recoded, err := json.Marshal(test.res)
				require.NoError(t, err)
				require.DeepEqual(t, []byte(test.result), recoded, "Unexpected recoded value")
			} else {
				assert.ErrorContains(t, test.err.Error(), err)
			}
		})
	}
}
