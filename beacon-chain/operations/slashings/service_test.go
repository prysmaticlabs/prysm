package slashings

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
)

var (
	_ = PoolManager(&Pool{})
	_ = PoolInserter(&Pool{})
	_ = PoolManager(&PoolMock{})
	_ = PoolInserter(&PoolMock{})
)

func TestPool_validatorSlashingPreconditionCheck_requiresLock(t *testing.T) {
	p := &Pool{}
	_, err := p.validatorSlashingPreconditionCheck(nil, 0)
	require.ErrorContains(t, "caller must hold read/write lock", err)
}
