//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/assert"
)

func TestBeaconApiHelpers_ValidRootCorrectFormat(t *testing.T) {
	const root = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	valid := validRoot(root)
	assert.Equal(t, true, valid)
}

func TestBeaconApiHelpers_ValidRootTooSmall(t *testing.T) {
	const root = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f"
	valid := validRoot(root)
	assert.Equal(t, false, valid)
}

func TestBeaconApiHelpers_ValidRootTooBig(t *testing.T) {
	const root = "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f22"
	valid := validRoot(root)
	assert.Equal(t, false, valid)
}

func TestBeaconApiHelpers_ValidRootEmpty(t *testing.T) {
	const root = ""
	valid := validRoot(root)
	assert.Equal(t, false, valid)
}

func TestBeaconApiHelpers_ValidRootNoPrefix(t *testing.T) {
	const root = "cf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	valid := validRoot(root)
	assert.Equal(t, false, valid)
}

func TestBeaconApiHelpers_InvalidCharacters(t *testing.T) {
	const root = "0xzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
	valid := validRoot(root)
	assert.Equal(t, false, valid)
}
