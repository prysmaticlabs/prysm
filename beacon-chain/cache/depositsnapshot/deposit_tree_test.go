package depositsnapshot

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/assert"
)

func Test_AddToEmptyDepositTree(t *testing.T) {
	dt := &DepositTree{
		tree:                    &ZeroNode{depth: DepositContractDepth},
		depositCount:            0,
		finalizedExecutionblock: [32]byte{},
	}
	newDt := NewDepositTree()
	assert.Equal(t, dt.tree.GetRoot(), newDt.tree.GetRoot())
	err := newDt.AddDeposit(hexString(t, fmt.Sprintf("%064d", 1)), uint64(1))
	assert.NoError(t, err)
	assert.NotEqual(t, dt.tree.GetRoot(), newDt.tree.GetRoot())
}
