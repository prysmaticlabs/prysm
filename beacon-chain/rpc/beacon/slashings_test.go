package beacon

import (
	"context"
	"testing"

	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestServer_SubmitProposerSlashing(t *testing.T) {
	ctx := context.Background()

	st, privs := testutil.DeterministicGenesisState(t, 64)
	slashedVal, err := st.ValidatorAtIndex(5)
	require.NoError(t, err)
	// We mark the validator at index 5 as already slashed.
	slashedVal.Slashed = true
	require.NoError(t, st.UpdateValidatorAtIndex(5, slashedVal))

	mb := &mockp2p.MockBroadcaster{}
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		SlashingsPool: slashings.NewPool(),
		Broadcaster:   mb,
	}

	// We want a proposer slashing for validator with index 2 to
	// be included in the pool.
	slashing, err := testutil.GenerateProposerSlashingForValidator(st, privs[2], uint64(2))
	require.NoError(t, err)

	_, err = bs.SubmitProposerSlashing(ctx, slashing)
	require.NoError(t, err)
	assert.Equal(t, true, mb.BroadcastCalled, "Expected broadcast to be called")
}

func TestServer_SubmitAttesterSlashing(t *testing.T) {
	ctx := context.Background()
	// We mark the validators at index 5, 6 as already slashed.
	st, privs := testutil.DeterministicGenesisState(t, 64)
	slashedVal, err := st.ValidatorAtIndex(5)
	require.NoError(t, err)

	// We mark the validator at index 5 as already slashed.
	slashedVal.Slashed = true
	require.NoError(t, st.UpdateValidatorAtIndex(5, slashedVal))

	mb := &mockp2p.MockBroadcaster{}
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		SlashingsPool: slashings.NewPool(),
		Broadcaster:   mb,
	}

	slashing, err := testutil.GenerateAttesterSlashingForValidator(st, privs[2], uint64(2))
	require.NoError(t, err)

	// We want the intersection of the slashing attesting indices
	// to be slashed, so we expect validators 2 and 3 to be in the response
	// slashed indices.
	_, err = bs.SubmitAttesterSlashing(ctx, slashing)
	require.NoError(t, err)
	assert.Equal(t, true, mb.BroadcastCalled, "Expected broadcast to be called when flag is set")
}
