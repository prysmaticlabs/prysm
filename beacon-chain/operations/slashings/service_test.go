package slashings

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/slashings/mock"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

var (
	_ = PoolManager(&Pool{})
	_ = PoolInserter(&Pool{})
	_ = PoolManager(&mock.PoolMock{})
	_ = PoolInserter(&mock.PoolMock{})
)

func TestPool_validatorSlashingPreconditionCheck_requiresLock(t *testing.T) {
	p := &Pool{}
	_, err := p.validatorSlashingPreconditionCheck(nil, 0)
	require.ErrorContains(t, "caller must hold read/write lock", err)
}
