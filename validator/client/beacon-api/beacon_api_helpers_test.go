//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/assert"
)

func TestHexValidation32Bytes(t *testing.T) {
	testCases := []struct {
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
			name:  "too small",
			input: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f",
			valid: false,
		},
		{
			name:  "too big",
			input: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f22",
			valid: false,
		},
		{
			name:  "empty",
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

	testSuites := []struct {
		name               string
		validationFunction func(string) bool
	}{
		{
			"validRoot",
			validRoot,
		},
	}

	for _, testSuite := range testSuites {
		for _, testCase := range testCases {
			t.Run(testSuite.name+" "+testCase.name, func(t *testing.T) {
				assert.Equal(t, testCase.valid, testSuite.validationFunction(testCase.input))
			})
		}
	}

}

func TestHexValidation4Bytes(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "correct format",
			input: "0x01234567",
			valid: true,
		},
		{
			name:  "too small",
			input: "0x0123456",
			valid: false,
		},
		{
			name:  "too big",
			input: "0x012345678",
			valid: false,
		},
		{
			name:  "empty",
			input: "",
			valid: false,
		},
		{
			name:  "no 0x prefix",
			input: "01234567",
			valid: false,
		},
		{
			name:  "invalid characters",
			input: "0xzzzzzzzz",
			valid: false,
		},
	}

	testSuites := []struct {
		name               string
		validationFunction func(string) bool
	}{
		{
			"validForkVersion",
			validForkVersion,
		},
		{
			"validDomainTypeVersion",
			validDomainTypeVersion,
		},
	}

	for _, testSuite := range testSuites {
		for _, testCase := range testCases {
			t.Run(testSuite.name+" "+testCase.name, func(t *testing.T) {
				assert.Equal(t, testCase.valid, testSuite.validationFunction(testCase.input))
			})
		}
	}
}
