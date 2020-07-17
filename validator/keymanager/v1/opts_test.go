package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
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
				if err != nil {
					t.Fatalf("Unexepcted error: %v", err)
				}
				recoded, err := json.Marshal(test.res)
				if err != nil {
					t.Fatalf("Unexepcted error encoding result: %v", err)
				}
				if !bytes.Equal([]byte(test.result), recoded) {
					t.Fatalf("Unexpected recoded value: expected %s, received %s", test.result, string(recoded))
				}

			} else {
				if err == nil {
					t.Fatalf("Missing expected error: %v", test.err)
				}
				if test.err.Error() != err.Error() {
					t.Fatalf("Unexpected error value: expected %v, received %v", test.err.Error(), err.Error())
				}
			}
		})
	}
}
