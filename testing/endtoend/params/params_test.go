package params

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func Test_port(t *testing.T) {
	var existingRegistrations []int

	p, err := port(2000, 3, 0, &existingRegistrations)
	require.NoError(t, err)
	assert.Equal(t, 2000, p)
	p, err = port(2000, 3, 1, &existingRegistrations)
	require.NoError(t, err)
	assert.Equal(t, 2016, p)
	p, err = port(2000, 3, 2, &existingRegistrations)
	require.NoError(t, err)
	assert.Equal(t, 2032, p)
	_, err = port(2000, 3, 2, &existingRegistrations)
	assert.NotNil(t, err)
	// We pass the last unavailable port
	_, err = port(2047, 3, 0, &existingRegistrations)
	assert.NotNil(t, err)
}
