//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"net/url"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/assert"
)

func TestBeaconApiHelpers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "correct format",
			input: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			valid: true,
		},
		{
			name:  "root too small",
			input: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f",
			valid: false,
		},
		{
			name:  "root too big",
			input: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f22",
			valid: false,
		},
		{
			name:  "empty root",
			input: "",
			valid: false,
		},
		{
			name:  "no 0x prefix",
			input: "cf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
			valid: false,
		},
		{
			name:  "invalid characters",
			input: "0xzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, validRoot(tt.input))
		})
	}
}

func TestBuildURL_NoParams(t *testing.T) {
	wanted := "/aaa/bbb/ccc"
	actual := buildURL("/aaa/bbb/ccc")
	assert.Equal(t, wanted, actual)
}

func TestBuildURL_WithParams(t *testing.T) {
	params := url.Values{}
	params.Add("xxxx", "1")
	params.Add("yyyy", "2")
	params.Add("zzzz", "3")

	wanted := "/aaa/bbb/ccc?xxxx=1&yyyy=2&zzzz=3"
	actual := buildURL("/aaa/bbb/ccc", params)
	assert.Equal(t, wanted, actual)
}
