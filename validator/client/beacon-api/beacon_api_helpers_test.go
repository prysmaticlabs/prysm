//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"math"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
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

func TestGetForkVersion(t *testing.T) {
	testSuites := []struct {
		name            string
		firstEpoch      types.Epoch
		lastEpoch       types.Epoch
		forkVersion     []byte
		mockForkVersion func(newVersion []byte)
	}{
		{
			name:            "genesis",
			firstEpoch:      0,
			lastEpoch:       9,
			forkVersion:     params.BeaconConfig().GenesisForkVersion,
			mockForkVersion: func(newVersion []byte) { params.BeaconConfig().GenesisForkVersion = newVersion },
		},
		{
			name:            "altair",
			firstEpoch:      10,
			lastEpoch:       19,
			forkVersion:     params.BeaconConfig().AltairForkVersion,
			mockForkVersion: func(newVersion []byte) { params.BeaconConfig().AltairForkVersion = newVersion },
		},
		{
			name:            "bellatrix",
			firstEpoch:      20,
			lastEpoch:       29,
			forkVersion:     params.BeaconConfig().BellatrixForkVersion,
			mockForkVersion: func(newVersion []byte) { params.BeaconConfig().BellatrixForkVersion = newVersion },
		},
		{
			name:            "capella",
			firstEpoch:      30,
			lastEpoch:       39,
			forkVersion:     params.BeaconConfig().CapellaForkVersion,
			mockForkVersion: func(newVersion []byte) { params.BeaconConfig().CapellaForkVersion = newVersion },
		},

		{
			name:            "sharding",
			firstEpoch:      40,
			lastEpoch:       math.MaxUint64,
			forkVersion:     params.BeaconConfig().ShardingForkVersion,
			mockForkVersion: func(newVersion []byte) { params.BeaconConfig().ShardingForkVersion = newVersion },
		},
	}

	testCases := []struct {
		name                string
		forkVersionOverride []byte
		valid               bool
	}{
		{
			name:  "valid",
			valid: true,
		},
		{
			name:                "empty version",
			forkVersionOverride: []byte{},
			valid:               false,
		},
		{
			name:                "version too small",
			forkVersionOverride: []byte{0, 0, 0},
			valid:               false,
		},
		{
			name:                "version too big",
			forkVersionOverride: []byte{0, 0, 0, 0, 0},
			valid:               false,
		},
	}

	for _, testSuite := range testSuites {
		for _, testCase := range testCases {
			t.Run(testSuite.name+" "+testCase.name, func(t *testing.T) {
				// Revert the beacon config to its previous state when we're done with the test case
				prevBeaconConfig := params.BeaconConfig().Copy()
				defer params.OverrideBeaconConfig(prevBeaconConfig)
				params.BeaconConfig().GenesisEpoch = 0
				params.BeaconConfig().AltairForkEpoch = 10
				params.BeaconConfig().BellatrixForkEpoch = 20
				params.BeaconConfig().CapellaForkEpoch = 30
				params.BeaconConfig().ShardingForkEpoch = 40

				if testCase.valid {
					forkVersion, err := getForkVersion(testSuite.firstEpoch)
					assert.NoError(t, err)
					assert.DeepEqual(t, forkVersion[:], testSuite.forkVersion)

					forkVersion, err = getForkVersion(testSuite.lastEpoch)
					assert.NoError(t, err)
					assert.DeepEqual(t, forkVersion[:], testSuite.forkVersion)
				} else {
					testSuite.mockForkVersion(testCase.forkVersionOverride)

					_, err := getForkVersion(testSuite.firstEpoch)
					assert.ErrorContains(t, "invalid fork version", err)
				}
			})
		}
	}
}
