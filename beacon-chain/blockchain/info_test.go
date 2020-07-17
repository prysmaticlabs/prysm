package blockchain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestService_TreeHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/tree", nil)
	require.NoError(t, err)

	ctx := context.Background()
	db, sCache := testDB.SetupDB(t)
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetBalances([]uint64{params.BeaconConfig().GweiPerEth}))
	cfg := &Config{
		BeaconDB: db,
		ForkChoiceStore: protoarray.New(
			0, // justifiedEpoch
			0, // finalizedEpoch
			[32]byte{'a'},
		),
		StateGen: stategen.New(db, sCache),
	}
	s, err := NewService(ctx, cfg)
	require.NoError(t, s.forkChoiceStore.ProcessBlock(ctx, 0, [32]byte{'a'}, [32]byte{'g'}, [32]byte{'c'}, 0, 0))
	require.NoError(t, s.forkChoiceStore.ProcessBlock(ctx, 1, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'c'}, 0, 0))
	s.setHead([32]byte{'a'}, testutil.NewBeaconBlock(), headState)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.TreeHandler)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
