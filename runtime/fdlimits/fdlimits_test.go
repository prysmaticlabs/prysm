package fdlimits_test

import (
	"syscall"
	"testing"

	"github.com/prysmaticlabs/prysm/runtime/fdlimits"
	"github.com/prysmaticlabs/prysm/testing/assert"
)

func TestSetMaxFdLimits(t *testing.T) {
	var limit syscall.Rlimit
	assert.NoError(t, syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit))
	wantedMax := limit.Max

	// Set it to a low value.
	limit.Cur = 2000
	assert.NoError(t, syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit))
	limit = syscall.Rlimit{}

	// Double check it works
	assert.NoError(t, syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit))
	assert.Equal(t, uint64(2000), limit.Cur)

	assert.NoError(t, fdlimits.SetMaxFdLimits())
	// Retrieve fd limit again.
	assert.NoError(t, syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit))

	assert.Equal(t, wantedMax, limit.Cur)
	assert.NotEqual(t, 2000, limit.Cur)
}
