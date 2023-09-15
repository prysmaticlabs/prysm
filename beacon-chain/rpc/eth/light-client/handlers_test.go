package lightclient

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/mux"
	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	testDB "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
)

func TestLightClientHandler_GetLightClientBootstrap(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	slot := primitives.Slot(params.BeaconConfig().AltairForkEpoch * primitives.Epoch(params.BeaconConfig().SlotsPerEpoch)).Add(1)

	db := testDB.SetupDB(t)
	b := util.NewBeaconBlockCapella()
	b.Block.StateRoot = bytesutil.PadTo([]byte("foo"), 32)
	b.Block.Slot = slot

	signedBlock, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	header, err := signedBlock.Header()
	require.NoError(t, err)

	util.SaveBlock(t, ctx, db, b)
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)

	bs, err := util.NewBeaconStateCapella(func(state *ethpb.BeaconStateCapella) error {
		state.BlockRoots[0] = r[:]
		return nil
	})
	require.NoError(t, err)

	require.NoError(t, bs.SetSlot(slot))
	require.NoError(t, bs.SetLatestBlockHeader(header.Header))

	require.NoError(t, db.SaveStateSummary(ctx, &ethpb.StateSummary{Root: r[:]}))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, r))
	require.NoError(t, db.SaveState(ctx, bs, r))

	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot: bs,
		}},
		BeaconDB:    db,
		HeadFetcher: mockChainService,
	}
	muxVars := make(map[string]string)
	muxVars["block_root"] = hexutil.Encode(r[:])
	request := httptest.NewRequest("GET", "http://foo.com/", nil)
	request = mux.SetURLVars(request, muxVars)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientBootstrap(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)
	resp := &LightClientBootstrapResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	require.Equal(t, "CAPELLA", resp.Version)
	require.Equal(t, hexutil.Encode(header.Header.BodyRoot), resp.Data.Header.BodyRoot)
}
