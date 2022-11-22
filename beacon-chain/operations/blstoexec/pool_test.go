package blstoexec

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	doublylinkedlist "github.com/prysmaticlabs/prysm/v3/container/doubly-linked-list"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestPendingBLSToExecChanges(t *testing.T) {
	t.Run("empty pool", func(t *testing.T) {
		pool := NewPool()
		changes, err := pool.PendingBLSToExecChanges()
		require.NoError(t, err)
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
		changes, err := pool.PendingBLSToExecChanges()
		require.NoError(t, err)
		assert.Equal(t, 2, len(changes))
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
		changes, err := pool.BLSToExecChangesForInclusion()
		require.NoError(t, err)
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
		changes, err := pool.BLSToExecChangesForInclusion()
		require.NoError(t, err)
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
		changes, err := pool.BLSToExecChangesForInclusion()
		require.NoError(t, err)
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
		require.Equal(t, 1, pool.pending.Len())
		require.Equal(t, 1, len(pool.m))
		n, ok := pool.m[0]
		require.Equal(t, true, ok)
		v, err := n.Value()
		require.NoError(t, err)
		assert.DeepEqual(t, change, v)
	})
	t.Run("item in pool", func(t *testing.T) {
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
		require.Equal(t, 2, pool.pending.Len())
		require.Equal(t, 2, len(pool.m))
		n, ok := pool.m[0]
		require.Equal(t, true, ok)
		v, err := n.Value()
		require.NoError(t, err)
		assert.DeepEqual(t, old, v)
		n, ok = pool.m[1]
		require.Equal(t, true, ok)
		v, err = n.Value()
		require.NoError(t, err)
		assert.DeepEqual(t, change, v)
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
		assert.Equal(t, 1, pool.pending.Len())
		require.Equal(t, 1, len(pool.m))
		n, ok := pool.m[0]
		require.Equal(t, true, ok)
		v, err := n.Value()
		require.NoError(t, err)
		assert.DeepEqual(t, old, v)
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
		require.NoError(t, pool.MarkIncluded(change))
		assert.Equal(t, 0, pool.pending.Len())
		assert.Equal(t, (*doublylinkedlist.Node[*eth.SignedBLSToExecutionChange])(nil), pool.m[0])
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
		require.NoError(t, pool.MarkIncluded(first))
		require.Equal(t, 2, pool.pending.Len())
		assert.Equal(t, (*doublylinkedlist.Node[*eth.SignedBLSToExecutionChange])(nil), pool.m[0])
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
		require.NoError(t, pool.MarkIncluded(third))
		require.Equal(t, 2, pool.pending.Len())
		assert.Equal(t, (*doublylinkedlist.Node[*eth.SignedBLSToExecutionChange])(nil), pool.m[2])
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
		require.NoError(t, pool.MarkIncluded(second))
		require.Equal(t, 2, pool.pending.Len())
		assert.Equal(t, (*doublylinkedlist.Node[*eth.SignedBLSToExecutionChange])(nil), pool.m[1])
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
		require.NoError(t, pool.MarkIncluded(change))
		require.Equal(t, 2, pool.pending.Len())
		assert.NotNil(t, pool.m[0])
		assert.NotNil(t, pool.m[1])
	})
}
