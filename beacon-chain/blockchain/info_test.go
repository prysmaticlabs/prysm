package blockchain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestService_TreeHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/tree", nil)
	require.NoError(t, err)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	headState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetBalances([]uint64{params.BeaconConfig().GweiPerEth}))
	cfg := &config{
		BeaconDB: beaconDB,
		ForkChoiceStore: protoarray.New(
			0, // justifiedEpoch
			0, // finalizedEpoch
			[32]byte{'a'},
		),
		StateGen: stategen.New(beaconDB),
	}
	s, err := NewService(ctx)
	require.NoError(t, err)
	s.cfg = cfg
	require.NoError(t, s.cfg.ForkChoiceStore.ProcessBlock(ctx, 0, [32]byte{'a'}, [32]byte{'g'}, [32]byte{'c'}, 0, 0))
	require.NoError(t, s.cfg.ForkChoiceStore.ProcessBlock(ctx, 1, [32]byte{'b'}, [32]byte{'a'}, [32]byte{'c'}, 0, 0))
	s.setHead([32]byte{'a'}, wrapper.WrappedPhase0SignedBeaconBlock(util.NewBeaconBlock()), headState)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.TreeHandler)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
