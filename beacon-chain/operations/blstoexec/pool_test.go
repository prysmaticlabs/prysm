package blstoexec

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestPendingBLSToExecChanges(t *testing.T) {
	t.Run("empty pool", func(t *testing.T) {
		pool := NewPool()
		changes := pool.PendingBLSToExecChanges()
		assert.Equal(t, 0, len(changes))
	})
	t.Run("non-empty pool", func(t *testing.T) {
		pool := NewPool()
		pool.InsertBLSToExecChange(&eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: 0,
			},
		})
		pool.InsertBLSToExecChange(&eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: 1,
			},
		})
		assert.Equal(t, 2, len(pool.PendingBLSToExecChanges()))
	})
}

func TestBLSToExecChangesForInclusion(t *testing.T) {
	t.Run("empty pool", func(t *testing.T) {
		pool := NewPool()
		for i := uint64(0); i < params.BeaconConfig().MaxBlsToExecutionChanges-1; i++ {
			pool.InsertBLSToExecChange(&eth.SignedBLSToExecutionChange{
				Message: &eth.BLSToExecutionChange{
					ValidatorIndex: types.ValidatorIndex(i),
				},
			})
		}
		changes := pool.BLSToExecChangesForInclusion()
		assert.Equal(t, int(params.BeaconConfig().MaxBlsToExecutionChanges-1), len(changes))
	})
	t.Run("MaxBlsToExecutionChanges in pool", func(t *testing.T) {
		pool := NewPool()
		for i := uint64(0); i < params.BeaconConfig().MaxBlsToExecutionChanges; i++ {
			pool.InsertBLSToExecChange(&eth.SignedBLSToExecutionChange{
				Message: &eth.BLSToExecutionChange{
					ValidatorIndex: types.ValidatorIndex(i),
				},
			})
		}
		changes := pool.BLSToExecChangesForInclusion()
		assert.Equal(t, int(params.BeaconConfig().MaxBlsToExecutionChanges), len(changes))
	})
	t.Run("more than MaxBlsToExecutionChanges in pool", func(t *testing.T) {
		pool := NewPool()
		for i := uint64(0); i < params.BeaconConfig().MaxBlsToExecutionChanges+1; i++ {
			pool.InsertBLSToExecChange(&eth.SignedBLSToExecutionChange{
				Message: &eth.BLSToExecutionChange{
					ValidatorIndex: types.ValidatorIndex(i),
				},
			})
		}
		changes := pool.BLSToExecChangesForInclusion()
		// We want FIFO semantics, which means validator with index 16 shouldn't be returned
		assert.Equal(t, int(params.BeaconConfig().MaxBlsToExecutionChanges), len(changes))
		for _, ch := range changes {
			assert.NotEqual(t, types.ValidatorIndex(16), ch.Message.ValidatorIndex)
		}
	})
}

func TestInsertBLSToExecChange(t *testing.T) {
	t.Run("empty pool", func(t *testing.T) {
		pool := NewPool()
		change := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(0),
			},
		}
		pool.InsertBLSToExecChange(change)
		require.Equal(t, 1, pool.pending.len)
		require.NotNil(t, pool.pending.first)
		assert.DeepEqual(t, change, pool.pending.first.value)
		require.NotNil(t, pool.pending.last)
		assert.DeepEqual(t, change, pool.pending.last.value)
		require.Equal(t, 1, len(pool.m))
		n, ok := pool.m[0]
		require.Equal(t, true, ok)
		assert.DeepEqual(t, change, n.value)
	})
	t.Run("one item in pool", func(t *testing.T) {
		pool := NewPool()
		old := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(0),
			},
		}
		change := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(1),
			},
		}
		pool.InsertBLSToExecChange(old)
		pool.InsertBLSToExecChange(change)
		require.Equal(t, 2, pool.pending.len)
		require.NotNil(t, pool.pending.first)
		assert.DeepEqual(t, old, pool.pending.first.value)
		require.NotNil(t, pool.pending.first.next)
		assert.Equal(t, change, pool.pending.first.next.value)
		require.NotNil(t, pool.pending.last)
		assert.DeepEqual(t, change, pool.pending.last.value)
		require.NotNil(t, pool.pending.last.prev)
		assert.Equal(t, old, pool.pending.last.prev.value)
		require.Equal(t, 2, len(pool.m))
		n, ok := pool.m[0]
		require.Equal(t, true, ok)
		assert.DeepEqual(t, old, n.value)
		n, ok = pool.m[1]
		require.Equal(t, true, ok)
		assert.DeepEqual(t, change, n.value)
	})
	t.Run("multiple items in pool", func(t *testing.T) {
		pool := NewPool()
		first := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(0),
			},
		}
		second := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(1),
			},
		}
		change := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(2),
			},
		}
		pool.InsertBLSToExecChange(first)
		pool.InsertBLSToExecChange(second)
		pool.InsertBLSToExecChange(change)
		require.Equal(t, 3, pool.pending.len)
		require.NotNil(t, pool.pending.first)
		assert.DeepEqual(t, first, pool.pending.first.value)
		require.NotNil(t, pool.pending.first.next)
		assert.Equal(t, second, pool.pending.first.next.value)
		assert.Equal(t, first, pool.pending.first.next.prev.value)
		require.NotNil(t, pool.pending.last)
		assert.DeepEqual(t, change, pool.pending.last.value)
		require.NotNil(t, pool.pending.first.next.next)
		assert.DeepEqual(t, change, pool.pending.first.next.next.value)
		require.NotNil(t, pool.pending.last.prev)
		assert.Equal(t, second, pool.pending.last.prev.value)
		require.Equal(t, 3, len(pool.m))
		n, ok := pool.m[0]
		require.Equal(t, true, ok)
		assert.DeepEqual(t, first, n.value)
		n, ok = pool.m[1]
		require.Equal(t, true, ok)
		assert.DeepEqual(t, second, n.value)
		n, ok = pool.m[2]
		require.Equal(t, true, ok)
		assert.DeepEqual(t, change, n.value)
	})
	t.Run("validator index already exists", func(t *testing.T) {
		pool := NewPool()
		old := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(0),
			},
			Signature: []byte("old"),
		}
		change := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(0),
			},
			Signature: []byte("change"),
		}
		pool.InsertBLSToExecChange(old)
		pool.InsertBLSToExecChange(change)
		assert.Equal(t, 1, pool.pending.len)
		require.Equal(t, 1, len(pool.m))
		n, ok := pool.m[0]
		require.Equal(t, true, ok)
		assert.DeepEqual(t, old, n.value)
	})
}

func TestMarkIncluded(t *testing.T) {
	t.Run("one element in pool", func(t *testing.T) {
		pool := NewPool()
		change := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(0),
			}}
		pool.InsertBLSToExecChange(change)
		pool.MarkIncluded(change)
		assert.Equal(t, 0, pool.pending.len)
		assert.Equal(t, (*listNode)(nil), pool.pending.first)
		assert.Equal(t, (*listNode)(nil), pool.pending.last)
		assert.Equal(t, (*listNode)(nil), pool.m[0])
	})
	t.Run("first of multiple elements", func(t *testing.T) {
		pool := NewPool()
		first := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(0),
			}}
		second := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(1),
			}}
		third := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(2),
			}}
		pool.InsertBLSToExecChange(first)
		pool.InsertBLSToExecChange(second)
		pool.InsertBLSToExecChange(third)
		pool.MarkIncluded(first)
		require.Equal(t, 2, pool.pending.len)
		require.NotNil(t, pool.pending.first)
		assert.Equal(t, second, pool.pending.first.value)
		assert.Equal(t, (*listNode)(nil), pool.pending.first.prev)
		assert.Equal(t, (*listNode)(nil), pool.m[0])
	})
	t.Run("last of multiple elements", func(t *testing.T) {
		pool := NewPool()
		first := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(0),
			}}
		second := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(1),
			}}
		third := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(2),
			}}
		pool.InsertBLSToExecChange(first)
		pool.InsertBLSToExecChange(second)
		pool.InsertBLSToExecChange(third)
		pool.MarkIncluded(third)
		require.Equal(t, 2, pool.pending.len)
		require.NotNil(t, pool.pending.last)
		assert.Equal(t, second, pool.pending.last.value)
		assert.Equal(t, (*listNode)(nil), pool.pending.last.next)
		assert.Equal(t, (*listNode)(nil), pool.m[2])
	})
	t.Run("in the middle of multiple elements", func(t *testing.T) {
		pool := NewPool()
		first := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(0),
			}}
		second := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(1),
			}}
		third := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(2),
			}}
		pool.InsertBLSToExecChange(first)
		pool.InsertBLSToExecChange(second)
		pool.InsertBLSToExecChange(third)
		pool.MarkIncluded(second)
		require.Equal(t, 2, pool.pending.len)
		require.NotNil(t, pool.pending.first)
		require.NotNil(t, pool.pending.last)
		assert.Equal(t, first, pool.pending.first.value)
		assert.Equal(t, third, pool.pending.last.value)
		assert.Equal(t, third, pool.pending.first.next.value)
		assert.Equal(t, first, pool.pending.last.prev.value)
		assert.Equal(t, (*listNode)(nil), pool.m[1])
	})
	t.Run("not in pool", func(t *testing.T) {
		pool := NewPool()
		first := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(0),
			}}
		second := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(1),
			}}
		change := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex: types.ValidatorIndex(2),
			}}
		pool.InsertBLSToExecChange(first)
		pool.InsertBLSToExecChange(second)
		pool.MarkIncluded(change)
		require.Equal(t, 2, pool.pending.len)
		assert.NotNil(t, pool.m[0])
		assert.NotNil(t, pool.m[1])
	})
}
