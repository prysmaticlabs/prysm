package blstoexec

import (
	"sync"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestPendingBLSToExecChanges(t *testing.T) {
	pool := &Pool{
		lock:    sync.RWMutex{},
		pending: make([]*eth.SignedBLSToExecutionChange, params.BeaconConfig().MaxBlsToExecutionChanges*2),
	}
	for i := range pool.pending {
		pool.pending[i] = &eth.SignedBLSToExecutionChange{}
	}
	t.Run("return MaxBlsToExecutionChanges", func(t *testing.T) {
		changes := pool.PendingBLSToExecChanges(false)
		assert.Equal(t, int(params.BeaconConfig().MaxBlsToExecutionChanges), len(changes))
	})
	t.Run("return all", func(t *testing.T) {
		changes := pool.PendingBLSToExecChanges(true)
		assert.Equal(t, len(pool.pending), len(changes))
	})
}

func TestInsertBLSToExecChange(t *testing.T) {
	st, err := util.NewBeaconStateCapella()
	require.NoError(t, err)
	require.NoError(t, st.AppendValidator(&eth.Validator{
		WithdrawalCredentials: []byte{0},
	}))
	require.NoError(t, st.AppendValidator(&eth.Validator{
		WithdrawalCredentials: []byte{1},
	}))
	pubkey := bytesutil.PadTo([]byte("pubkey"), fieldparams.BLSPubkeyLength)
	address := bytesutil.PadTo([]byte("address"), fieldparams.ExecutionAddressLength)
	sig := bytesutil.PadTo([]byte("sig"), fieldparams.BLSSignatureLength)

	t.Run("ok", func(t *testing.T) {
		pool := NewPool()
		change := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex:     0,
				FromBlsPubkey:      pubkey,
				ToExecutionAddress: address,
			},
			Signature: sig,
		}
		pool.InsertBLSToExecChange(st, change)
		pending := pool.PendingBLSToExecChanges(true)
		require.Equal(t, 1, len(pending))
		assert.DeepEqual(t, change, pending[0])
	})
	t.Run("nil change", func(t *testing.T) {
		pool := NewPool()
		pool.InsertBLSToExecChange(st, nil)
		assert.Equal(t, 0, len(pool.PendingBLSToExecChanges(true)))
	})
	t.Run("nil message", func(t *testing.T) {
		pool := NewPool()
		pool.InsertBLSToExecChange(st, &eth.SignedBLSToExecutionChange{
			Message:   nil,
			Signature: sig,
		})
		assert.Equal(t, 0, len(pool.PendingBLSToExecChanges(true)))
	})
	t.Run("wrong signature length", func(t *testing.T) {
		pool := NewPool()
		pool.InsertBLSToExecChange(st, &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex:     0,
				FromBlsPubkey:      pubkey,
				ToExecutionAddress: address,
			},
			Signature: make([]byte, 50),
		})
		assert.Equal(t, 0, len(pool.PendingBLSToExecChanges(true)))
	})
	t.Run("wrong pubkey length", func(t *testing.T) {
		pool := NewPool()
		pool.InsertBLSToExecChange(st, &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex:     0,
				FromBlsPubkey:      make([]byte, 50),
				ToExecutionAddress: address,
			},
			Signature: sig,
		})
		assert.Equal(t, 0, len(pool.PendingBLSToExecChanges(true)))
	})
	t.Run("wrong address length", func(t *testing.T) {
		pool := NewPool()
		pool.InsertBLSToExecChange(st, &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex:     0,
				FromBlsPubkey:      pubkey,
				ToExecutionAddress: make([]byte, 50),
			},
			Signature: sig,
		})
		assert.Equal(t, 0, len(pool.PendingBLSToExecChanges(true)))
	})
	t.Run("change for validator already exists", func(t *testing.T) {
		pool := NewPool()
		change := &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex:     0,
				FromBlsPubkey:      pubkey,
				ToExecutionAddress: address,
			},
			Signature: sig,
		}
		// Insert twice
		pool.InsertBLSToExecChange(st, change)
		pool.InsertBLSToExecChange(st, change)
		assert.Equal(t, 1, len(pool.PendingBLSToExecChanges(true)))
	})
	t.Run("validator already has ETH1 creds", func(t *testing.T) {
		pool := NewPool()
		pool.InsertBLSToExecChange(st, &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex:     1,
				FromBlsPubkey:      pubkey,
				ToExecutionAddress: address,
			},
			Signature: sig,
		})
		assert.Equal(t, 0, len(pool.PendingBLSToExecChanges(true)))
	})
}

func TestMarkIncluded(t *testing.T) {
	pool := &Pool{
		lock:    sync.RWMutex{},
		pending: make([]*eth.SignedBLSToExecutionChange, 3),
	}
	change0 := &eth.SignedBLSToExecutionChange{Message: &eth.BLSToExecutionChange{ValidatorIndex: 0}}
	change1 := &eth.SignedBLSToExecutionChange{Message: &eth.BLSToExecutionChange{ValidatorIndex: 1}}
	change2 := &eth.SignedBLSToExecutionChange{Message: &eth.BLSToExecutionChange{ValidatorIndex: 2}}
	pool.pending[0] = change0
	pool.pending[1] = change1
	pool.pending[2] = change2
	pool.MarkIncluded(change1)
	pending := pool.PendingBLSToExecChanges(true)
	require.Equal(t, 2, len(pending))
	assert.DeepEqual(t, change0, pending[0])
	assert.DeepEqual(t, change2, pending[1])
}
