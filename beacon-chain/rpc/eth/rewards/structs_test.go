package rewards_test

import (
	"encoding/json"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/rewards"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestMarshalBlockRewards(t *testing.T) {
	r := &rewards.BlockRewards{
		ProposerIndex:     123,
		Total:             123,
		Attestations:      123,
		SyncAggregate:     123,
		ProposerSlashings: 123,
		AttesterSlashings: 123,
	}
	j, err := json.Marshal(r)
	require.NoError(t, err)
	expected := `{"proposer_index":"123","total":"123","attestations":"123","sync_aggregate":"123","proposer_slashings":"123","attester_slashings":"123"}`
	assert.Equal(t, expected, string(j))
}

func TestUnmarshalBlockRewards(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		s := `{"proposer_index":"123","total":"123","attestations":"123","sync_aggregate":"123","proposer_slashings":"123","attester_slashings":"123"}`
		r := &rewards.BlockRewards{}
		require.NoError(t, r.UnmarshalJSON([]byte(s)))
		expected := &rewards.BlockRewards{
			ProposerIndex:     123,
			Total:             123,
			Attestations:      123,
			SyncAggregate:     123,
			ProposerSlashings: 123,
			AttesterSlashings: 123,
		}
		assert.DeepEqual(t, expected, r)
	})
	t.Run("invalid proposer index", func(t *testing.T) {
		s := `{"proposer_index":"foo","total":"123","attestations":"123","sync_aggregate":"123","proposer_slashings":"123","attester_slashings":"123"}`
		r := &rewards.BlockRewards{}
		err := r.UnmarshalJSON([]byte(s))
		assert.ErrorContains(t, "could not unmarshal proposer index", err)
	})
	t.Run("invalid total", func(t *testing.T) {
		s := `{"proposer_index":"123","total":"foo","attestations":"123","sync_aggregate":"123","proposer_slashings":"123","attester_slashings":"123"}`
		r := &rewards.BlockRewards{}
		err := r.UnmarshalJSON([]byte(s))
		assert.ErrorContains(t, "could not unmarshal total", err)
	})
	t.Run("invalid attestations", func(t *testing.T) {
		s := `{"proposer_index":"123","total":"123","attestations":"foo","sync_aggregate":"123","proposer_slashings":"123","attester_slashings":"123"}`
		r := &rewards.BlockRewards{}
		err := r.UnmarshalJSON([]byte(s))
		assert.ErrorContains(t, "could not unmarshal attestations", err)
	})
	t.Run("invalid sync aggregate", func(t *testing.T) {
		s := `{"proposer_index":"123","total":"123","attestations":"123","sync_aggregate":"foo","proposer_slashings":"123","attester_slashings":"123"}`
		r := &rewards.BlockRewards{}
		err := r.UnmarshalJSON([]byte(s))
		assert.ErrorContains(t, "could not unmarshal sync aggregate", err)
	})
	t.Run("invalid proposer slashings", func(t *testing.T) {
		s := `{"proposer_index":"123","total":"123","attestations":"123","sync_aggregate":"123","proposer_slashings":"foo","attester_slashings":"123"}`
		r := &rewards.BlockRewards{}
		err := r.UnmarshalJSON([]byte(s))
		assert.ErrorContains(t, "could not unmarshal proposer slashings", err)
	})
	t.Run("invalid attester slashings", func(t *testing.T) {
		s := `{"proposer_index":"123","total":"123","attestations":"123","sync_aggregate":"123","proposer_slashings":"123","attester_slashings":"foo"}`
		r := &rewards.BlockRewards{}
		err := r.UnmarshalJSON([]byte(s))
		assert.ErrorContains(t, "could not unmarshal attester slashings", err)
	})
}
