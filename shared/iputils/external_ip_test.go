package iputils_test

import (
	"regexp"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/iputils"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestExternalIPv4(t *testing.T) {
	// Regular expression format for IPv4
	IPv4Format := `\.\d{1,3}\.\d{1,3}\b`
	test, err := iputils.ExternalIPv4()
	require.NoError(t, err)

	valid := regexp.MustCompile(IPv4Format)
	assert.Equal(t, true, valid.MatchString(test))
}
