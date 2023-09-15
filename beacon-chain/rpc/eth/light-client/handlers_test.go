package lightclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func TestLightClientHandler_GetLightClientUpdatesByRange(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateCapella()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	parent := util.NewBeaconBlockCapella()
	parent.Block.Slot = slot.Sub(1)

	signedParent, err := blocks.NewSignedBeaconBlock(parent)
	require.NoError(t, err)

	parentHeader, err := signedParent.Header()
	require.NoError(t, err)
	attestedHeader := parentHeader.Header

	err = attestedState.SetLatestBlockHeader(attestedHeader)
	require.NoError(t, err)
	attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
	require.NoError(t, err)

	// get a new signed block so the root is updated with the new state root
	parent.Block.StateRoot = attestedStateRoot[:]
	signedParent, err = blocks.NewSignedBeaconBlock(parent)
	require.NoError(t, err)

	st, err := util.NewBeaconStateCapella()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockCapella()
	block.Block.Slot = slot
	block.Block.ParentRoot = parentRoot[:]

	for i := uint64(0); i < config.SyncCommitteeSize; i++ {
		block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
	}

	signedBlock, err := blocks.NewSignedBeaconBlock(block)
	require.NoError(t, err)

	h, err := signedBlock.Header()
	require.NoError(t, err)

	err = st.SetLatestBlockHeader(h.Header)
	require.NoError(t, err)
	stateRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err)

	// get a new signed block so the root is updated with the new state root
	block.Block.StateRoot = stateRoot[:]
	signedBlock, err = blocks.NewSignedBeaconBlock(block)
	require.NoError(t, err)

	db := testDB.SetupDB(t)

	util.SaveBlock(t, ctx, db, block)
	util.SaveBlock(t, ctx, db, parent)
	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, db.SaveStateSummary(ctx, &ethpb.StateSummary{Root: root[:]}))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))
	require.NoError(t, db.SaveState(ctx, st, root))

	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		BeaconDB:    db,
		HeadFetcher: mockChainService,
	}
	startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
	url := fmt.Sprintf("http://foo.com/?count=1&start_period=%d", startPeriod)
	request := httptest.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientUpdatesByRange(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	resp := &LightClientUpdatesByRangeResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	require.Equal(t, 1, len(resp.Updates))
	require.Equal(t, "CAPELLA", resp.Updates[0].Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), resp.Updates[0].Data.AttestedHeader.BodyRoot)
}
