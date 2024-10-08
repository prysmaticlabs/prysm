package lightclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	mock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestLightClientHandler_GetLightClientBootstrap_Altair(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestAltair()

	slot := l.State.Slot()
	stateRoot, err := l.State.HashTreeRoot(l.Ctx)
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{BlockToReturn: l.Block}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot: l.State,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	request := httptest.NewRequest("GET", "http://foo.com/", nil)
	request.SetPathValue("block_root", hexutil.Encode(stateRoot[:]))
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientBootstrap(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientBootstrapResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Data.Header, &respHeader)
	require.NoError(t, err)
	require.Equal(t, "altair", resp.Version)

	blockHeader, err := l.Block.Header()
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(blockHeader.Header.BodyRoot), respHeader.Beacon.BodyRoot)
	require.Equal(t, strconv.FormatUint(uint64(blockHeader.Header.Slot), 10), respHeader.Beacon.Slot)

	require.NotNil(t, resp.Data.CurrentSyncCommittee)
	require.NotNil(t, resp.Data.CurrentSyncCommitteeBranch)

}

func TestLightClientHandler_GetLightClientBootstrap_Capella(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestCapella(false) // result is same for true and false

	slot := l.State.Slot()
	stateRoot, err := l.State.HashTreeRoot(l.Ctx)
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{BlockToReturn: l.Block}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot: l.State,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	request := httptest.NewRequest("GET", "http://foo.com/", nil)
	request.SetPathValue("block_root", hexutil.Encode(stateRoot[:]))
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientBootstrap(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientBootstrapResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Data.Header, &respHeader)
	require.NoError(t, err)
	require.Equal(t, "capella", resp.Version)

	blockHeader, err := l.Block.Header()
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(blockHeader.Header.BodyRoot), respHeader.Beacon.BodyRoot)
	require.Equal(t, strconv.FormatUint(uint64(blockHeader.Header.Slot), 10), respHeader.Beacon.Slot)

	require.NotNil(t, resp.Data.CurrentSyncCommittee)
	require.NotNil(t, resp.Data.CurrentSyncCommitteeBranch)
}

func TestLightClientHandler_GetLightClientBootstrap_Deneb(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestDeneb(false) // result is same for true and false

	slot := l.State.Slot()
	stateRoot, err := l.State.HashTreeRoot(l.Ctx)
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{BlockToReturn: l.Block}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot: l.State,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	request := httptest.NewRequest("GET", "http://foo.com/", nil)
	request.SetPathValue("block_root", hexutil.Encode(stateRoot[:]))
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientBootstrap(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientBootstrapResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Data.Header, &respHeader)
	require.NoError(t, err)
	require.Equal(t, "deneb", resp.Version)

	blockHeader, err := l.Block.Header()
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(blockHeader.Header.BodyRoot), respHeader.Beacon.BodyRoot)
	require.Equal(t, strconv.FormatUint(uint64(blockHeader.Header.Slot), 10), respHeader.Beacon.Slot)

	require.NotNil(t, resp.Data.CurrentSyncCommittee)
	require.NotNil(t, resp.Data.CurrentSyncCommitteeBranch)
}

func TestLightClientHandler_GetLightClientBootstrap_Electra(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestElectra(false) // result is same for true and false

	slot := l.State.Slot()
	stateRoot, err := l.State.HashTreeRoot(l.Ctx)
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{BlockToReturn: l.Block}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot: l.State,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	request := httptest.NewRequest("GET", "http://foo.com/", nil)
	request.SetPathValue("block_root", hexutil.Encode(stateRoot[:]))
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientBootstrap(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientBootstrapResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Data.Header, &respHeader)
	require.NoError(t, err)
	require.Equal(t, "electra", resp.Version)

	blockHeader, err := l.Block.Header()
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(blockHeader.Header.BodyRoot), respHeader.Beacon.BodyRoot)
	require.Equal(t, strconv.FormatUint(uint64(blockHeader.Header.Slot), 10), respHeader.Beacon.Slot)

	require.NotNil(t, resp.Data.CurrentSyncCommittee)
	require.NotNil(t, resp.Data.CurrentSyncCommitteeBranch)
}

func TestLightClientHandler_GetLightClientUpdatesByRangeAltair(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	parent := util.NewBeaconBlockAltair()
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

	st, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockAltair()
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
	url := fmt.Sprintf("http://foo.com/?count=1&start_period=%d", startPeriod)
	request := httptest.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientUpdatesByRange(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientUpdatesByRangeResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Updates[0].Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Updates))
	require.Equal(t, "altair", resp.Updates[0].Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp)
}

func TestLightClientHandler_GetLightClientUpdatesByRangeCapella(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
	url := fmt.Sprintf("http://foo.com/?count=1&start_period=%d", startPeriod)
	request := httptest.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientUpdatesByRange(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientUpdatesByRangeResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
	require.NoError(t, err)
	var respHeader structs.LightClientHeaderCapella
	err = json.Unmarshal(resp.Updates[0].Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Updates))
	require.Equal(t, "capella", resp.Updates[0].Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp)
}

func TestLightClientHandler_GetLightClientUpdatesByRangeDeneb(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.DenebForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateDeneb()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	parent := util.NewBeaconBlockDeneb()
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

	st, err := util.NewBeaconStateDeneb()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockDeneb()
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
	url := fmt.Sprintf("http://foo.com/?count=1&start_period=%d", startPeriod)
	request := httptest.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientUpdatesByRange(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientUpdatesByRangeResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
	require.NoError(t, err)
	var respHeader structs.LightClientHeaderDeneb
	err = json.Unmarshal(resp.Updates[0].Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Updates))
	require.Equal(t, "deneb", resp.Updates[0].Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp)
}

func TestLightClientHandler_GetLightClientUpdatesByRange_TooBigInputCountAltair(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	parent := util.NewBeaconBlockAltair()
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

	st, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockAltair()
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
	count := 129 // config.MaxRequestLightClientUpdates is 128
	url := fmt.Sprintf("http://foo.com/?count=%d&start_period=%d", count, startPeriod)
	request := httptest.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientUpdatesByRange(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientUpdatesByRangeResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Updates[0].Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Updates)) // Even with big count input, the response is still the max available period, which is 1 in test case.
	require.Equal(t, "altair", resp.Updates[0].Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp)
}

func TestLightClientHandler_GetLightClientUpdatesByRange_TooBigInputCountCapella(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
	count := 129 // config.MaxRequestLightClientUpdates is 128
	url := fmt.Sprintf("http://foo.com/?count=%d&start_period=%d", count, startPeriod)
	request := httptest.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientUpdatesByRange(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientUpdatesByRangeResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
	require.NoError(t, err)
	var respHeader structs.LightClientHeaderCapella
	err = json.Unmarshal(resp.Updates[0].Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Updates)) // Even with big count input, the response is still the max available period, which is 1 in test case.
	require.Equal(t, "capella", resp.Updates[0].Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp)
}

func TestLightClientHandler_GetLightClientUpdatesByRange_TooBigInputCountDeneb(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.DenebForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateDeneb()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	parent := util.NewBeaconBlockDeneb()
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

	st, err := util.NewBeaconStateDeneb()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockDeneb()
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
	count := 129 // config.MaxRequestLightClientUpdates is 128
	url := fmt.Sprintf("http://foo.com/?count=%d&start_period=%d", count, startPeriod)
	request := httptest.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientUpdatesByRange(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientUpdatesByRangeResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
	require.NoError(t, err)
	var respHeader structs.LightClientHeaderDeneb
	err = json.Unmarshal(resp.Updates[0].Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Updates)) // Even with big count input, the response is still the max available period, which is 1 in test case.
	require.Equal(t, "deneb", resp.Updates[0].Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp)
}

// TODO - check for not having any blocks from the min period, and startPeriod being too early
func TestLightClientHandler_GetLightClientUpdatesByRange_TooEarlyPeriodAltair(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	parent := util.NewBeaconBlockAltair()
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

	st, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockAltair()
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	startPeriod := 1 // very early period before Altair fork
	count := 1
	url := fmt.Sprintf("http://foo.com/?count=%d&start_period=%d", count, startPeriod)
	request := httptest.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientUpdatesByRange(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientUpdatesByRangeResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Updates[0].Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Updates))
	require.Equal(t, "altair", resp.Updates[0].Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp)
}

// TODO - same as above
func TestLightClientHandler_GetLightClientUpdatesByRange_TooBigCountAltair(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	parent := util.NewBeaconBlockAltair()
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

	st, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockAltair()
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	startPeriod := 1 // very early period before Altair fork
	count := 10      // This is big count as we only have one period in test case.
	url := fmt.Sprintf("http://foo.com/?count=%d&start_period=%d", count, startPeriod)
	request := httptest.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientUpdatesByRange(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp structs.LightClientUpdatesByRangeResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Updates[0].Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Updates))
	require.Equal(t, "altair", resp.Updates[0].Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp)
}

func TestLightClientHandler_GetLightClientUpdatesByRange_BeforeAltair(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Sub(1)

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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
	count := 1
	url := fmt.Sprintf("http://foo.com/?count=%d&start_period=%d", count, startPeriod)
	request := httptest.NewRequest("GET", url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientUpdatesByRange(writer, request)

	require.Equal(t, http.StatusNotFound, writer.Code)
}

func TestLightClientHandler_GetLightClientFinalityUpdateAltair(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: config.AltairForkEpoch - 10,
		Root:  make([]byte, 32),
	}))

	parent := util.NewBeaconBlockAltair()
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

	st, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockAltair()
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
		root: true,
	}}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	request := httptest.NewRequest("GET", "http://foo.com", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientFinalityUpdate(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp *structs.LightClientUpdateResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, "altair", resp.Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp.Data)
}

func TestLightClientHandler_GetLightClientFinalityUpdateCapella(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateCapella()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: config.AltairForkEpoch - 10,
		Root:  make([]byte, 32),
	}))

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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
		root: true,
	}}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	request := httptest.NewRequest("GET", "http://foo.com", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientFinalityUpdate(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp *structs.LightClientUpdateResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, "capella", resp.Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp.Data)
}

func TestLightClientHandler_GetLightClientFinalityUpdateDeneb(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.DenebForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateDeneb()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: config.AltairForkEpoch - 10,
		Root:  make([]byte, 32),
	}))

	parent := util.NewBeaconBlockDeneb()
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

	st, err := util.NewBeaconStateDeneb()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockDeneb()
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
		root: true,
	}}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	request := httptest.NewRequest("GET", "http://foo.com", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientFinalityUpdate(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp *structs.LightClientUpdateResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp)
	require.NoError(t, err)
	var respHeader structs.LightClientHeaderDeneb
	err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, "deneb", resp.Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp.Data)
}

func TestLightClientHandler_GetLightClientOptimisticUpdateAltair(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: config.AltairForkEpoch - 10,
		Root:  make([]byte, 32),
	}))

	parent := util.NewBeaconBlockAltair()
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

	st, err := util.NewBeaconStateAltair()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockAltair()
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
		root: true,
	}}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	request := httptest.NewRequest("GET", "http://foo.com", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientOptimisticUpdate(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp *structs.LightClientUpdateResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp)
	require.NoError(t, err)
	var respHeader structs.LightClientHeader
	err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, "altair", resp.Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp.Data)
}

func TestLightClientHandler_GetLightClientOptimisticUpdateCapella(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateCapella()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: config.AltairForkEpoch - 10,
		Root:  make([]byte, 32),
	}))

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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
		root: true,
	}}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	request := httptest.NewRequest("GET", "http://foo.com", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientOptimisticUpdate(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp *structs.LightClientUpdateResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp)
	require.NoError(t, err)
	var respHeader structs.LightClientHeaderCapella
	err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, "capella", resp.Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp.Data)
}

func TestLightClientHandler_GetLightClientOptimisticUpdateDeneb(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.DenebForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateDeneb()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: config.AltairForkEpoch - 10,
		Root:  make([]byte, 32),
	}))

	parent := util.NewBeaconBlockDeneb()
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

	st, err := util.NewBeaconStateDeneb()
	require.NoError(t, err)
	err = st.SetSlot(slot)
	require.NoError(t, err)

	parentRoot, err := signedParent.Block().HashTreeRoot()
	require.NoError(t, err)

	block := util.NewBeaconBlockDeneb()
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
		root: true,
	}}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}
	request := httptest.NewRequest("GET", "http://foo.com", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetLightClientOptimisticUpdate(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	var resp *structs.LightClientUpdateResponse
	err = json.Unmarshal(writer.Body.Bytes(), &resp)
	require.NoError(t, err)
	var respHeader structs.LightClientHeaderDeneb
	err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
	require.NoError(t, err)
	require.Equal(t, "deneb", resp.Version)
	require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
	require.NotNil(t, resp.Data)
}

func TestLightClientHandler_GetLightClientEventBlock(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateCapella()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: config.AltairForkEpoch - 10,
		Root:  make([]byte, 32),
	}))

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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
		root: true,
	}}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}

	minSignaturesRequired := uint64(100)
	eventBlock, err := s.suitableBlock(ctx, minSignaturesRequired)

	require.NoError(t, err)
	require.NotNil(t, eventBlock)
	require.Equal(t, slot, eventBlock.Block().Slot())
	syncAggregate, err := eventBlock.Block().Body().SyncAggregate()
	require.NoError(t, err)
	require.Equal(t, true, syncAggregate.SyncCommitteeBits.Count() >= minSignaturesRequired)
}

func TestLightClientHandler_GetLightClientEventBlock_NeedFetchParent(t *testing.T) {
	helpers.ClearCache()
	ctx := context.Background()
	config := params.BeaconConfig()
	slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)

	attestedState, err := util.NewBeaconStateCapella()
	require.NoError(t, err)
	err = attestedState.SetSlot(slot.Sub(1))
	require.NoError(t, err)

	require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: config.AltairForkEpoch - 10,
		Root:  make([]byte, 32),
	}))

	parent := util.NewBeaconBlockCapella()
	parent.Block.Slot = slot.Sub(1)
	for i := uint64(0); i < config.SyncCommitteeSize; i++ {
		parent.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
	}

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

	for i := uint64(0); i < 10; i++ {
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

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{
		RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
			parentRoot: signedParent,
			root:       signedBlock,
		},
		SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
			slot.Sub(1): signedParent,
			slot:        signedBlock,
		},
	}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
		root: true,
	}}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot.Sub(1): attestedState,
			slot:        st,
		}},
		Blocker:     mockBlocker,
		HeadFetcher: mockChainService,
	}

	minSignaturesRequired := uint64(100)
	eventBlock, err := s.suitableBlock(ctx, minSignaturesRequired)

	require.NoError(t, err)
	require.NotNil(t, eventBlock)
	syncAggregate, err := eventBlock.Block().Body().SyncAggregate()
	require.NoError(t, err)
	require.Equal(t, true, syncAggregate.SyncCommitteeBits.Count() >= minSignaturesRequired)
	require.Equal(t, slot-1, eventBlock.Block().Slot())
}
