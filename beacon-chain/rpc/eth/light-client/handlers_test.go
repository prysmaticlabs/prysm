package lightclient

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	mock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
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
	mockChainInfoFetcher := &mock.ChainService{Slot: &slot}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot: l.State,
		}},
		Blocker:          mockBlocker,
		HeadFetcher:      mockChainService,
		ChainInfoFetcher: mockChainInfoFetcher,
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

func TestLightClientHandler_GetLightClientBootstrap_Bellatrix(t *testing.T) {
	l := util.NewTestLightClient(t).SetupTestBellatrix()

	slot := l.State.Slot()
	stateRoot, err := l.State.HashTreeRoot(l.Ctx)
	require.NoError(t, err)

	mockBlocker := &testutil.MockBlocker{BlockToReturn: l.Block}
	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot}
	mockChainInfoFetcher := &mock.ChainService{Slot: &slot}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot: l.State,
		}},
		Blocker:          mockBlocker,
		HeadFetcher:      mockChainService,
		ChainInfoFetcher: mockChainInfoFetcher,
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
	require.Equal(t, "bellatrix", resp.Version)

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
	mockChainInfoFetcher := &mock.ChainService{Slot: &slot}
	s := &Server{
		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
			slot: l.State,
		}},
		Blocker:          mockBlocker,
		HeadFetcher:      mockChainService,
		ChainInfoFetcher: mockChainInfoFetcher,
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

//func TestLightClientHandler_GetLightClientBootstrap_Deneb(t *testing.T) {
//	l := util.NewTestLightClient(t).SetupTestDeneb(false) // result is same for true and false
//
//	slot := l.State.Slot()
//	stateRoot, err := l.State.HashTreeRoot(l.Ctx)
//	require.NoError(t, err)
//
//	mockBlocker := &testutil.MockBlocker{BlockToReturn: l.Block}
//	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot}
//	s := &Server{
//		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
//			slot: l.State,
//		}},
//		Blocker:     mockBlocker,
//		HeadFetcher: mockChainService,
//	}
//	request := httptest.NewRequest("GET", "http://foo.com/", nil)
//	request.SetPathValue("block_root", hexutil.Encode(stateRoot[:]))
//	writer := httptest.NewRecorder()
//	writer.Body = &bytes.Buffer{}
//
//	s.GetLightClientBootstrap(writer, request)
//	require.Equal(t, http.StatusOK, writer.Code)
//	var resp structs.LightClientBootstrapResponse
//	err = json.Unmarshal(writer.Body.Bytes(), &resp)
//	require.NoError(t, err)
//	var respHeader structs.LightClientHeader
//	err = json.Unmarshal(resp.Data.Header, &respHeader)
//	require.NoError(t, err)
//	require.Equal(t, "deneb", resp.Version)
//
//	blockHeader, err := l.Block.Header()
//	require.NoError(t, err)
//	require.Equal(t, hexutil.Encode(blockHeader.Header.BodyRoot), respHeader.Beacon.BodyRoot)
//	require.Equal(t, strconv.FormatUint(uint64(blockHeader.Header.Slot), 10), respHeader.Beacon.Slot)
//
//	require.NotNil(t, resp.Data.CurrentSyncCommittee)
//	require.NotNil(t, resp.Data.CurrentSyncCommitteeBranch)
//}

//func TestLightClientHandler_GetLightClientBootstrap_Electra(t *testing.T) {
//	l := util.NewTestLightClient(t).SetupTestElectra(false) // result is same for true and false
//
//	slot := l.State.Slot()
//	stateRoot, err := l.State.HashTreeRoot(l.Ctx)
//	require.NoError(t, err)
//
//	mockBlocker := &testutil.MockBlocker{BlockToReturn: l.Block}
//	mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot}
//	s := &Server{
//		Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
//			slot: l.State,
//		}},
//		Blocker:     mockBlocker,
//		HeadFetcher: mockChainService,
//	}
//	request := httptest.NewRequest("GET", "http://foo.com/", nil)
//	request.SetPathValue("block_root", hexutil.Encode(stateRoot[:]))
//	writer := httptest.NewRecorder()
//	writer.Body = &bytes.Buffer{}
//
//	s.GetLightClientBootstrap(writer, request)
//	require.Equal(t, http.StatusOK, writer.Code)
//	var resp structs.LightClientBootstrapResponse
//	err = json.Unmarshal(writer.Body.Bytes(), &resp)
//	require.NoError(t, err)
//	var respHeader structs.LightClientHeader
//	err = json.Unmarshal(resp.Data.Header, &respHeader)
//	require.NoError(t, err)
//	require.Equal(t, "electra", resp.Version)
//
//	blockHeader, err := l.Block.Header()
//	require.NoError(t, err)
//	require.Equal(t, hexutil.Encode(blockHeader.Header.BodyRoot), respHeader.Beacon.BodyRoot)
//	require.Equal(t, strconv.FormatUint(uint64(blockHeader.Header.Slot), 10), respHeader.Beacon.Slot)
//
//	require.NotNil(t, resp.Data.CurrentSyncCommittee)
//	require.NotNil(t, resp.Data.CurrentSyncCommitteeBranch)
//}

// GetLightClientByRange tests

//func TestLightClientHandler_GetLightClientUpdatesByRangeAltair(t *testing.T) {
//	helpers.ClearCache()
//	ctx := context.Background()
//
//	params.SetupTestConfigCleanup(t)
//	config := params.BeaconConfig()
//	config.AltairForkEpoch = 0
//	params.OverrideBeaconConfig(config)
//
//	slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//	st, err := util.NewBeaconStateAltair()
//	require.NoError(t, err)
//	err = st.SetSlot(slot)
//	require.NoError(t, err)
//
//	db := setupDB(t)
//
//	updatePeriod := uint64(slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch)))
//
//	updateRaw := &pb.LightClientUpdateAltair{
//		AttestedHeader: &pb.LightClientHeaderAltair{
//			Beacon: &pb.BeaconBlockHeader{
//				Slot:          slot.Sub(1),
//				ProposerIndex: 1,
//				ParentRoot:    []byte{1, 1, 1},
//				StateRoot:     []byte{1, 1, 1},
//				BodyRoot:      []byte{1, 1, 1},
//			},
//		},
//		NextSyncCommittee: &pb.SyncCommittee{
//			Pubkeys:         nil,
//			AggregatePubkey: nil,
//		},
//		NextSyncCommitteeBranch: nil,
//		FinalizedHeader:         nil,
//		FinalityBranch:          nil,
//		SyncAggregate: &pb.SyncAggregate{
//			SyncCommitteeBits:      []byte{1, 1, 1},
//			SyncCommitteeSignature: []byte{1, 1, 1},
//		},
//		SignatureSlot: 7,
//	}
//
//	update, err := light_client.NewWrappedUpdate(updateRaw)
//	require.NoError(t, err)
//	err = db.SaveLightClientUpdate(ctx, updatePeriod, update)
//	require.NoError(t, err)
//
//	mockChainService := &mock.ChainService{State: st}
//	s := &Server{
//		HeadFetcher: mockChainService,
//		BeaconDB:    db,
//	}
//	startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//	url := fmt.Sprintf("http://foo.com/?count=1&start_period=%d", startPeriod)
//	request := httptest.NewRequest("GET", url, nil)
//	writer := httptest.NewRecorder()
//	writer.Body = &bytes.Buffer{}
//
//	s.GetLightClientUpdatesByRange(writer, request)
//
//	require.Equal(t, http.StatusOK, writer.Code)
//	var resp structs.LightClientUpdatesByRangeResponse
//	err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//	require.NoError(t, err)
//	require.Equal(t, 1, len(resp.Updates))
//	require.Equal(t, "altair", resp.Updates[0].Version)
//	updateJson, err := structs.LightClientUpdateFromConsensus(update)
//	require.NoError(t, err)
//	require.DeepEqual(t, updateJson, resp.Updates[0].Data)
//}

//	func TestLightClientHandler_GetLightClientUpdatesByRangeCapella(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 0
//		config.CapellaForkEpoch = 1
//		params.OverrideBeaconConfig(config)
//		slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateCapella()
//		require.NoError(t, err)
//		err = st.SetSlot(slot)
//		require.NoError(t, err)
//
//		db := setupDB(t)
//
//		updatePeriod := uint64(slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch)))
//
//		update := &ethpbv2.LightClientUpdate{
//			AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
//					HeaderCapella: &ethpbv2.LightClientHeaderCapella{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          slot.Sub(1),
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//						Execution: &enginev1.ExecutionPayloadHeaderCapella{
//							FeeRecipient: []byte{1, 2, 3},
//						},
//						ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//					},
//				},
//			},
//			NextSyncCommittee: &ethpbv2.SyncCommittee{
//				Pubkeys:         nil,
//				AggregatePubkey: nil,
//			},
//			NextSyncCommitteeBranch: nil,
//			FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
//					HeaderCapella: &ethpbv2.LightClientHeaderCapella{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          12,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//						Execution: &enginev1.ExecutionPayloadHeaderCapella{
//							FeeRecipient: []byte{1, 2, 3},
//						},
//						ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//					},
//				},
//			},
//			FinalityBranch: nil,
//			SyncAggregate: &ethpbv1.SyncAggregate{
//				SyncCommitteeBits:      []byte{1, 1, 1},
//				SyncCommitteeSignature: []byte{1, 1, 1},
//			},
//			SignatureSlot: 7,
//		}
//
//		err = db.SaveLightClientUpdate(ctx, updatePeriod, &ethpbv2.LightClientUpdateWithVersion{
//			Version: version.Capella,
//			Data:    update,
//		})
//		require.NoError(t, err)
//
//		mockChainService := &mock.ChainService{State: st}
//		s := &Server{
//			HeadFetcher: mockChainService,
//			BeaconDB:    db,
//		}
//		startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//		url := fmt.Sprintf("http://foo.com/?count=1&start_period=%d", startPeriod)
//		request := httptest.NewRequest("GET", url, nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientUpdatesByRange(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp structs.LightClientUpdatesByRangeResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//		require.NoError(t, err)
//		require.Equal(t, 1, len(resp.Updates))
//		require.Equal(t, "capella", resp.Updates[0].Version)
//		updateJson, err := structs.LightClientUpdateFromConsensus(update)
//		require.NoError(t, err)
//		require.DeepEqual(t, updateJson, resp.Updates[0].Data)
//	}
//
//	func TestLightClientHandler_GetLightClientUpdatesByRangeDeneb(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 0
//		config.DenebForkEpoch = 1
//		params.OverrideBeaconConfig(config)
//		slot := primitives.Slot(config.DenebForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateDeneb()
//		require.NoError(t, err)
//		err = st.SetSlot(slot)
//		require.NoError(t, err)
//
//		db := setupDB(t)
//
//		updatePeriod := uint64(slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch)))
//
//		update := &ethpbv2.LightClientUpdate{
//			AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
//					HeaderCapella: &ethpbv2.LightClientHeaderCapella{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          slot.Sub(1),
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//						Execution: &enginev1.ExecutionPayloadHeaderCapella{
//							FeeRecipient: []byte{1, 2, 3},
//						},
//						ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//					},
//				},
//			},
//			NextSyncCommittee: &ethpbv2.SyncCommittee{
//				Pubkeys:         nil,
//				AggregatePubkey: nil,
//			},
//			NextSyncCommitteeBranch: nil,
//			FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderDeneb{
//					HeaderDeneb: &ethpbv2.LightClientHeaderDeneb{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          12,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//						Execution: &enginev1.ExecutionPayloadHeaderDeneb{
//							FeeRecipient: []byte{1, 2, 3},
//						},
//						ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//					},
//				},
//			},
//			FinalityBranch: nil,
//			SyncAggregate: &ethpbv1.SyncAggregate{
//				SyncCommitteeBits:      []byte{1, 1, 1},
//				SyncCommitteeSignature: []byte{1, 1, 1},
//			},
//			SignatureSlot: 7,
//		}
//
//		err = db.SaveLightClientUpdate(ctx, updatePeriod, &ethpbv2.LightClientUpdateWithVersion{
//			Version: version.Deneb,
//			Data:    update,
//		})
//		require.NoError(t, err)
//
//		mockChainService := &mock.ChainService{State: st}
//		s := &Server{
//			HeadFetcher: mockChainService,
//			BeaconDB:    db,
//		}
//		startPeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//		url := fmt.Sprintf("http://foo.com/?count=1&start_period=%d", startPeriod)
//		request := httptest.NewRequest("GET", url, nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientUpdatesByRange(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp structs.LightClientUpdatesByRangeResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//		require.NoError(t, err)
//		require.Equal(t, 1, len(resp.Updates))
//		require.Equal(t, "deneb", resp.Updates[0].Version)
//		updateJson, err := structs.LightClientUpdateFromConsensus(update)
//		require.NoError(t, err)
//		require.DeepEqual(t, updateJson, resp.Updates[0].Data)
//	}
//
//	func TestLightClientHandler_GetLightClientUpdatesByRangeMultipleAltair(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 0
//		config.EpochsPerSyncCommitteePeriod = 1
//		params.OverrideBeaconConfig(config)
//		slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		headSlot := slot.Add(33) // 2 periods
//		err = st.SetSlot(headSlot)
//		require.NoError(t, err)
//
//		db := setupDB(t)
//
//		updates := make([]*ethpbv2.LightClientUpdate, 2)
//
//		updatePeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//		for i := 0; i < 2; i++ {
//			newSlot := slot.Add(uint64(i))
//
//			update := &ethpbv2.LightClientUpdate{
//				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
//						HeaderAltair: &ethpbv2.LightClientHeader{
//							Beacon: &ethpbv1.BeaconBlockHeader{
//								Slot:          newSlot,
//								ProposerIndex: 1,
//								ParentRoot:    []byte{1, 1, 1},
//								StateRoot:     []byte{1, 1, 1},
//								BodyRoot:      []byte{1, 1, 1},
//							},
//						},
//					},
//				},
//				NextSyncCommittee: &ethpbv2.SyncCommittee{
//					Pubkeys:         nil,
//					AggregatePubkey: nil,
//				},
//				NextSyncCommitteeBranch: nil,
//				FinalizedHeader:         nil,
//				FinalityBranch:          nil,
//				SyncAggregate: &ethpbv1.SyncAggregate{
//					SyncCommitteeBits:      []byte{1, 1, 1},
//					SyncCommitteeSignature: []byte{1, 1, 1},
//				},
//				SignatureSlot: 7,
//			}
//
//			updates[i] = update
//
//			err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//				Version: version.Altair,
//				Data:    update,
//			})
//			require.NoError(t, err)
//
//			updatePeriod++
//		}
//
//		mockChainService := &mock.ChainService{State: st}
//		s := &Server{
//			HeadFetcher: mockChainService,
//			BeaconDB:    db,
//		}
//		startPeriod := slot.Sub(1).Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//		url := fmt.Sprintf("http://foo.com/?count=100&start_period=%d", startPeriod)
//		request := httptest.NewRequest("GET", url, nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientUpdatesByRange(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp structs.LightClientUpdatesByRangeResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//		require.NoError(t, err)
//		require.Equal(t, 2, len(resp.Updates))
//		for i, update := range updates {
//			require.Equal(t, "altair", resp.Updates[i].Version)
//			updateJson, err := structs.LightClientUpdateFromConsensus(update)
//			require.NoError(t, err)
//			require.DeepEqual(t, updateJson, resp.Updates[i].Data)
//		}
//	}
//
//	func TestLightClientHandler_GetLightClientUpdatesByRangeMultipleCapella(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 0
//		config.CapellaForkEpoch = 1
//		config.EpochsPerSyncCommitteePeriod = 1
//		params.OverrideBeaconConfig(config)
//		slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		headSlot := slot.Add(33) // 2 periods
//		err = st.SetSlot(headSlot)
//		require.NoError(t, err)
//
//		db := setupDB(t)
//
//		updates := make([]*ethpbv2.LightClientUpdate, 2)
//
//		updatePeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//		for i := 0; i < 2; i++ {
//
//			newSlot := slot.Add(uint64(i))
//
//			update := &ethpbv2.LightClientUpdate{
//				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//					Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
//						HeaderCapella: &ethpbv2.LightClientHeaderCapella{
//							Beacon: &ethpbv1.BeaconBlockHeader{
//								Slot:          newSlot,
//								ProposerIndex: 1,
//								ParentRoot:    []byte{1, 1, 1},
//								StateRoot:     []byte{1, 1, 1},
//								BodyRoot:      []byte{1, 1, 1},
//							},
//							Execution: &enginev1.ExecutionPayloadHeaderCapella{
//								FeeRecipient: []byte{1, 2, 3},
//							},
//							ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//						},
//					},
//				},
//				NextSyncCommittee: &ethpbv2.SyncCommittee{
//					Pubkeys:         nil,
//					AggregatePubkey: nil,
//				},
//				NextSyncCommitteeBranch: nil,
//				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
//					Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
//						HeaderCapella: &ethpbv2.LightClientHeaderCapella{
//							Beacon: &ethpbv1.BeaconBlockHeader{
//								Slot:          12,
//								ProposerIndex: 1,
//								ParentRoot:    []byte{1, 1, 1},
//								StateRoot:     []byte{1, 1, 1},
//								BodyRoot:      []byte{1, 1, 1},
//							},
//							Execution: &enginev1.ExecutionPayloadHeaderCapella{
//								FeeRecipient: []byte{1, 2, 3},
//							},
//							ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//						},
//					},
//				},
//				FinalityBranch: nil,
//				SyncAggregate: &ethpbv1.SyncAggregate{
//					SyncCommitteeBits:      []byte{1, 1, 1},
//					SyncCommitteeSignature: []byte{1, 1, 1},
//				},
//				SignatureSlot: 7,
//			}
//
//			updates[i] = update
//
//			err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//				Version: version.Capella,
//				Data:    update,
//			})
//			require.NoError(t, err)
//
//			updatePeriod++
//		}
//
//		mockChainService := &mock.ChainService{State: st}
//		s := &Server{
//			HeadFetcher: mockChainService,
//			BeaconDB:    db,
//		}
//		startPeriod := slot.Sub(1).Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//		url := fmt.Sprintf("http://foo.com/?count=100&start_period=%d", startPeriod)
//		request := httptest.NewRequest("GET", url, nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientUpdatesByRange(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp structs.LightClientUpdatesByRangeResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//		require.NoError(t, err)
//		require.Equal(t, 2, len(resp.Updates))
//		for i, update := range updates {
//			require.Equal(t, "capella", resp.Updates[i].Version)
//			updateJson, err := structs.LightClientUpdateFromConsensus(update)
//			require.NoError(t, err)
//			require.DeepEqual(t, updateJson, resp.Updates[i].Data)
//		}
//	}
//
//	func TestLightClientHandler_GetLightClientUpdatesByRangeMultipleDeneb(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 0
//		config.DenebForkEpoch = 1
//		config.EpochsPerSyncCommitteePeriod = 1
//		params.OverrideBeaconConfig(config)
//		slot := primitives.Slot(config.DenebForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		headSlot := slot.Add(33)
//		err = st.SetSlot(headSlot)
//		require.NoError(t, err)
//
//		db := setupDB(t)
//
//		updates := make([]*ethpbv2.LightClientUpdate, 2)
//
//		updatePeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//		for i := 0; i < 2; i++ {
//
//			newSlot := slot.Add(uint64(i))
//
//			update := &ethpbv2.LightClientUpdate{
//				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//					Header: &ethpbv2.LightClientHeaderContainer_HeaderDeneb{
//						HeaderDeneb: &ethpbv2.LightClientHeaderDeneb{
//							Beacon: &ethpbv1.BeaconBlockHeader{
//								Slot:          newSlot,
//								ProposerIndex: 1,
//								ParentRoot:    []byte{1, 1, 1},
//								StateRoot:     []byte{1, 1, 1},
//								BodyRoot:      []byte{1, 1, 1},
//							},
//							Execution: &enginev1.ExecutionPayloadHeaderDeneb{
//								FeeRecipient: []byte{1, 2, 3},
//							},
//							ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//						},
//					},
//				},
//				NextSyncCommittee: &ethpbv2.SyncCommittee{
//					Pubkeys:         nil,
//					AggregatePubkey: nil,
//				},
//				NextSyncCommitteeBranch: nil,
//				FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
//					Header: &ethpbv2.LightClientHeaderContainer_HeaderDeneb{
//						HeaderDeneb: &ethpbv2.LightClientHeaderDeneb{
//							Beacon: &ethpbv1.BeaconBlockHeader{
//								Slot:          12,
//								ProposerIndex: 1,
//								ParentRoot:    []byte{1, 1, 1},
//								StateRoot:     []byte{1, 1, 1},
//								BodyRoot:      []byte{1, 1, 1},
//							},
//							Execution: &enginev1.ExecutionPayloadHeaderDeneb{
//								FeeRecipient: []byte{1, 2, 3},
//							},
//							ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//						},
//					},
//				},
//				FinalityBranch: nil,
//				SyncAggregate: &ethpbv1.SyncAggregate{
//					SyncCommitteeBits:      []byte{1, 1, 1},
//					SyncCommitteeSignature: []byte{1, 1, 1},
//				},
//				SignatureSlot: 7,
//			}
//
//			updates[i] = update
//
//			err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//				Version: version.Deneb,
//				Data:    update,
//			})
//			require.NoError(t, err)
//
//			updatePeriod++
//		}
//
//		mockChainService := &mock.ChainService{State: st}
//		s := &Server{
//			HeadFetcher: mockChainService,
//			BeaconDB:    db,
//		}
//		startPeriod := slot.Sub(1).Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//		url := fmt.Sprintf("http://foo.com/?count=100&start_period=%d", startPeriod)
//		request := httptest.NewRequest("GET", url, nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientUpdatesByRange(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp structs.LightClientUpdatesByRangeResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//		require.NoError(t, err)
//		require.Equal(t, 2, len(resp.Updates))
//		for i, update := range updates {
//			require.Equal(t, "deneb", resp.Updates[i].Version)
//			updateJson, err := structs.LightClientUpdateFromConsensus(update)
//			require.NoError(t, err)
//			require.DeepEqual(t, updateJson, resp.Updates[i].Data)
//		}
//	}
//
//	func TestLightClientHandler_GetLightClientUpdatesByRangeMultipleForksAltairCapella(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 0
//		config.CapellaForkEpoch = 1
//		config.EpochsPerSyncCommitteePeriod = 1
//		params.OverrideBeaconConfig(config)
//		slotCapella := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//		slotAltair := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		headSlot := slotCapella.Add(1)
//		err = st.SetSlot(headSlot)
//		require.NoError(t, err)
//
//		db := setupDB(t)
//
//		updates := make([]*ethpbv2.LightClientUpdate, 2)
//
//		updatePeriod := slotAltair.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//		update := &ethpbv2.LightClientUpdate{
//			AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
//					HeaderAltair: &ethpbv2.LightClientHeader{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          slotAltair,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//					},
//				},
//			},
//			NextSyncCommittee: &ethpbv2.SyncCommittee{
//				Pubkeys:         nil,
//				AggregatePubkey: nil,
//			},
//			NextSyncCommitteeBranch: nil,
//			FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
//					HeaderAltair: &ethpbv2.LightClientHeader{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          12,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//					},
//				},
//			},
//			FinalityBranch: nil,
//			SyncAggregate: &ethpbv1.SyncAggregate{
//				SyncCommitteeBits:      []byte{1, 1, 1},
//				SyncCommitteeSignature: []byte{1, 1, 1},
//			},
//			SignatureSlot: slotAltair,
//		}
//
//		updates[0] = update
//
//		err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//			Version: version.Altair,
//			Data:    update,
//		})
//		require.NoError(t, err)
//
//		updatePeriod = slotCapella.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//		update = &ethpbv2.LightClientUpdate{
//			AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
//					HeaderCapella: &ethpbv2.LightClientHeaderCapella{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          slotCapella,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//						Execution: &enginev1.ExecutionPayloadHeaderCapella{
//							FeeRecipient: []byte{1, 2, 3},
//						},
//						ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//					},
//				},
//			},
//			NextSyncCommittee: &ethpbv2.SyncCommittee{
//				Pubkeys:         nil,
//				AggregatePubkey: nil,
//			},
//			NextSyncCommitteeBranch: nil,
//			FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
//					HeaderCapella: &ethpbv2.LightClientHeaderCapella{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          12,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//						Execution: &enginev1.ExecutionPayloadHeaderCapella{
//							FeeRecipient: []byte{1, 2, 3},
//						},
//						ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//					},
//				},
//			},
//			FinalityBranch: nil,
//			SyncAggregate: &ethpbv1.SyncAggregate{
//				SyncCommitteeBits:      []byte{1, 1, 1},
//				SyncCommitteeSignature: []byte{1, 1, 1},
//			},
//			SignatureSlot: slotCapella,
//		}
//
//		updates[1] = update
//
//		err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//			Version: version.Capella,
//			Data:    update,
//		})
//		require.NoError(t, err)
//
//		mockChainService := &mock.ChainService{State: st}
//		s := &Server{
//			HeadFetcher: mockChainService,
//			BeaconDB:    db,
//		}
//		startPeriod := 0
//		url := fmt.Sprintf("http://foo.com/?count=100&start_period=%d", startPeriod)
//		request := httptest.NewRequest("GET", url, nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientUpdatesByRange(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp structs.LightClientUpdatesByRangeResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//		require.NoError(t, err)
//		require.Equal(t, 2, len(resp.Updates))
//		for i, update := range updates {
//			if i < 1 {
//				require.Equal(t, "altair", resp.Updates[i].Version)
//			} else {
//				require.Equal(t, "capella", resp.Updates[i].Version)
//			}
//			updateJson, err := structs.LightClientUpdateFromConsensus(update)
//			require.NoError(t, err)
//			require.DeepEqual(t, updateJson, resp.Updates[i].Data)
//		}
//	}
//
//	func TestLightClientHandler_GetLightClientUpdatesByRangeMultipleForksCapellaDeneb(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 0
//		config.CapellaForkEpoch = 1
//		config.DenebForkEpoch = 2
//		config.EpochsPerSyncCommitteePeriod = 1
//		params.OverrideBeaconConfig(config)
//		slotDeneb := primitives.Slot(config.DenebForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//		slotCapella := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		headSlot := slotDeneb.Add(1)
//		err = st.SetSlot(headSlot)
//		require.NoError(t, err)
//
//		db := setupDB(t)
//
//		updates := make([]*ethpbv2.LightClientUpdate, 2)
//
//		updatePeriod := slotCapella.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//		update := &ethpbv2.LightClientUpdate{
//			AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
//					HeaderCapella: &ethpbv2.LightClientHeaderCapella{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          slotCapella,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//						Execution: &enginev1.ExecutionPayloadHeaderCapella{
//							FeeRecipient: []byte{1, 2, 3},
//						},
//						ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//					},
//				},
//			},
//			NextSyncCommittee: &ethpbv2.SyncCommittee{
//				Pubkeys:         nil,
//				AggregatePubkey: nil,
//			},
//			NextSyncCommitteeBranch: nil,
//			FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderCapella{
//					HeaderCapella: &ethpbv2.LightClientHeaderCapella{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          12,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//						Execution: &enginev1.ExecutionPayloadHeaderCapella{
//							FeeRecipient: []byte{1, 2, 3},
//						},
//						ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//					},
//				},
//			},
//			FinalityBranch: nil,
//			SyncAggregate: &ethpbv1.SyncAggregate{
//				SyncCommitteeBits:      []byte{1, 1, 1},
//				SyncCommitteeSignature: []byte{1, 1, 1},
//			},
//			SignatureSlot: slotCapella,
//		}
//
//		updates[0] = update
//
//		err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//			Version: version.Capella,
//			Data:    update,
//		})
//		require.NoError(t, err)
//
//		updatePeriod = slotDeneb.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//		update = &ethpbv2.LightClientUpdate{
//			AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderDeneb{
//					HeaderDeneb: &ethpbv2.LightClientHeaderDeneb{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          slotDeneb,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//						Execution: &enginev1.ExecutionPayloadHeaderDeneb{
//							FeeRecipient: []byte{1, 2, 3},
//						},
//						ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//					},
//				},
//			},
//			NextSyncCommittee: &ethpbv2.SyncCommittee{
//				Pubkeys:         nil,
//				AggregatePubkey: nil,
//			},
//			NextSyncCommitteeBranch: nil,
//			FinalizedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderDeneb{
//					HeaderDeneb: &ethpbv2.LightClientHeaderDeneb{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          12,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//						Execution: &enginev1.ExecutionPayloadHeaderDeneb{
//							FeeRecipient: []byte{1, 2, 3},
//						},
//						ExecutionBranch: [][]byte{{1, 2, 3}, {4, 5, 6}},
//					},
//				},
//			},
//			FinalityBranch: nil,
//			SyncAggregate: &ethpbv1.SyncAggregate{
//				SyncCommitteeBits:      []byte{1, 1, 1},
//				SyncCommitteeSignature: []byte{1, 1, 1},
//			},
//			SignatureSlot: slotDeneb,
//		}
//
//		updates[1] = update
//
//		err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//			Version: version.Deneb,
//			Data:    update,
//		})
//		require.NoError(t, err)
//
//		mockChainService := &mock.ChainService{State: st}
//		s := &Server{
//			HeadFetcher: mockChainService,
//			BeaconDB:    db,
//		}
//		startPeriod := 1
//		url := fmt.Sprintf("http://foo.com/?count=100&start_period=%d", startPeriod)
//		request := httptest.NewRequest("GET", url, nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientUpdatesByRange(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp structs.LightClientUpdatesByRangeResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//		require.NoError(t, err)
//		require.Equal(t, 2, len(resp.Updates))
//		for i, update := range updates {
//			if i < 1 {
//				require.Equal(t, "capella", resp.Updates[i].Version)
//			} else {
//				require.Equal(t, "deneb", resp.Updates[i].Version)
//			}
//			updateJson, err := structs.LightClientUpdateFromConsensus(update)
//			require.NoError(t, err)
//			require.DeepEqual(t, updateJson, resp.Updates[i].Data)
//		}
//	}
//
//	func TestLightClientHandler_GetLightClientUpdatesByRangeCountBiggerThanLimit(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 0
//		config.EpochsPerSyncCommitteePeriod = 1
//		config.MaxRequestLightClientUpdates = 2
//		params.OverrideBeaconConfig(config)
//		slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		headSlot := slot.Add(65)
//		err = st.SetSlot(headSlot)
//		require.NoError(t, err)
//
//		db := setupDB(t)
//
//		updates := make([]*ethpbv2.LightClientUpdate, 3)
//
//		updatePeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//		for i := 0; i < 3; i++ {
//			newSlot := slot.Add(uint64(i * 32))
//
//			update := &ethpbv2.LightClientUpdate{
//				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
//						HeaderAltair: &ethpbv2.LightClientHeader{
//							Beacon: &ethpbv1.BeaconBlockHeader{
//								Slot:          newSlot,
//								ProposerIndex: 1,
//								ParentRoot:    []byte{1, 1, 1},
//								StateRoot:     []byte{1, 1, 1},
//								BodyRoot:      []byte{1, 1, 1},
//							},
//						},
//					},
//				},
//				NextSyncCommittee: &ethpbv2.SyncCommittee{
//					Pubkeys:         nil,
//					AggregatePubkey: nil,
//				},
//				NextSyncCommitteeBranch: nil,
//				FinalizedHeader:         nil,
//				FinalityBranch:          nil,
//				SyncAggregate: &ethpbv1.SyncAggregate{
//					SyncCommitteeBits:      []byte{1, 1, 1},
//					SyncCommitteeSignature: []byte{1, 1, 1},
//				},
//				SignatureSlot: 7,
//			}
//
//			updates[i] = update
//
//			err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//				Version: version.Altair,
//				Data:    update,
//			})
//			require.NoError(t, err)
//
//			updatePeriod++
//		}
//
//		mockChainService := &mock.ChainService{State: st}
//		s := &Server{
//			HeadFetcher: mockChainService,
//			BeaconDB:    db,
//		}
//		startPeriod := 0
//		url := fmt.Sprintf("http://foo.com/?count=4&start_period=%d", startPeriod)
//		request := httptest.NewRequest("GET", url, nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientUpdatesByRange(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp structs.LightClientUpdatesByRangeResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//		require.NoError(t, err)
//		require.Equal(t, 2, len(resp.Updates))
//		for i, update := range updates {
//			if i < 2 {
//				require.Equal(t, "altair", resp.Updates[i].Version)
//				updateJson, err := structs.LightClientUpdateFromConsensus(update)
//				require.NoError(t, err)
//				require.DeepEqual(t, updateJson, resp.Updates[i].Data)
//			}
//		}
//	}
//
//	func TestLightClientHandler_GetLightClientUpdatesByRangeCountBiggerThanMax(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 0
//		config.EpochsPerSyncCommitteePeriod = 1
//		config.MaxRequestLightClientUpdates = 2
//		params.OverrideBeaconConfig(config)
//		slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		headSlot := slot.Add(65)
//		err = st.SetSlot(headSlot)
//		require.NoError(t, err)
//
//		db := setupDB(t)
//
//		updates := make([]*ethpbv2.LightClientUpdate, 3)
//
//		updatePeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//		for i := 0; i < 3; i++ {
//			newSlot := slot.Add(uint64(i * 32))
//
//			update := &ethpbv2.LightClientUpdate{
//				AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//					Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
//						HeaderAltair: &ethpbv2.LightClientHeader{
//							Beacon: &ethpbv1.BeaconBlockHeader{
//								Slot:          newSlot,
//								ProposerIndex: 1,
//								ParentRoot:    []byte{1, 1, 1},
//								StateRoot:     []byte{1, 1, 1},
//								BodyRoot:      []byte{1, 1, 1},
//							},
//						},
//					},
//				},
//				NextSyncCommittee: &ethpbv2.SyncCommittee{
//					Pubkeys:         nil,
//					AggregatePubkey: nil,
//				},
//				NextSyncCommitteeBranch: nil,
//				FinalizedHeader:         nil,
//				FinalityBranch:          nil,
//				SyncAggregate: &ethpbv1.SyncAggregate{
//					SyncCommitteeBits:      []byte{1, 1, 1},
//					SyncCommitteeSignature: []byte{1, 1, 1},
//				},
//				SignatureSlot: 7,
//			}
//
//			updates[i] = update
//			err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//				Version: version.Altair,
//				Data:    update,
//			})
//			require.NoError(t, err)
//
//			updatePeriod++
//		}
//
//		mockChainService := &mock.ChainService{State: st}
//		s := &Server{
//			HeadFetcher: mockChainService,
//			BeaconDB:    db,
//		}
//		startPeriod := 0
//		url := fmt.Sprintf("http://foo.com/?count=10&start_period=%d", startPeriod)
//		request := httptest.NewRequest("GET", url, nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientUpdatesByRange(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp structs.LightClientUpdatesByRangeResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//		require.NoError(t, err)
//		require.Equal(t, 2, len(resp.Updates))
//		for i, update := range updates {
//			if i < 2 {
//				require.Equal(t, "altair", resp.Updates[i].Version)
//				updateJson, err := structs.LightClientUpdateFromConsensus(update)
//				require.NoError(t, err)
//				require.DeepEqual(t, updateJson, resp.Updates[i].Data)
//			}
//		}
//	}
//
//	func TestLightClientHandler_GetLightClientUpdatesByRangeStartPeriodBeforeAltair(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 1
//		config.EpochsPerSyncCommitteePeriod = 1
//		params.OverrideBeaconConfig(config)
//		slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		headSlot := slot.Add(1)
//		err = st.SetSlot(headSlot)
//		require.NoError(t, err)
//
//		db := setupDB(t)
//
//		updatePeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//		update := &ethpbv2.LightClientUpdate{
//			AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//				Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
//					HeaderAltair: &ethpbv2.LightClientHeader{
//						Beacon: &ethpbv1.BeaconBlockHeader{
//							Slot:          slot,
//							ProposerIndex: 1,
//							ParentRoot:    []byte{1, 1, 1},
//							StateRoot:     []byte{1, 1, 1},
//							BodyRoot:      []byte{1, 1, 1},
//						},
//					},
//				},
//			},
//			NextSyncCommittee: &ethpbv2.SyncCommittee{
//				Pubkeys:         nil,
//				AggregatePubkey: nil,
//			},
//			NextSyncCommitteeBranch: nil,
//			FinalizedHeader:         nil,
//			FinalityBranch:          nil,
//			SyncAggregate: &ethpbv1.SyncAggregate{
//				SyncCommitteeBits:      []byte{1, 1, 1},
//				SyncCommitteeSignature: []byte{1, 1, 1},
//			},
//			SignatureSlot: 7,
//		}
//
//		err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//			Version: version.Altair,
//			Data:    update,
//		})
//		require.NoError(t, err)
//
//		mockChainService := &mock.ChainService{State: st}
//		s := &Server{
//			HeadFetcher: mockChainService,
//			BeaconDB:    db,
//		}
//		startPeriod := 0
//		url := fmt.Sprintf("http://foo.com/?count=2&start_period=%d", startPeriod)
//		request := httptest.NewRequest("GET", url, nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientUpdatesByRange(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp structs.LightClientUpdatesByRangeResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//		require.NoError(t, err)
//		require.Equal(t, 1, len(resp.Updates))
//
//		require.Equal(t, "altair", resp.Updates[0].Version)
//		updateJson, err := structs.LightClientUpdateFromConsensus(update)
//		require.NoError(t, err)
//		require.DeepEqual(t, updateJson, resp.Updates[0].Data)
//
// }
//
//	func TestLightClientHandler_GetLightClientUpdatesByRangeMissingUpdates(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		config.AltairForkEpoch = 0
//		config.EpochsPerSyncCommitteePeriod = 1
//		params.OverrideBeaconConfig(config)
//		slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		headSlot := slot.Add(65)
//		err = st.SetSlot(headSlot)
//		require.NoError(t, err)
//
//		t.Run("missing update in the middle", func(t *testing.T) {
//			db := setupDB(t)
//
//			updates := make([]*ethpbv2.LightClientUpdate, 3)
//
//			updatePeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//			for i := 0; i < 3; i++ {
//				if i == 1 { // skip this update
//					updatePeriod++
//					continue
//				}
//				newSlot := slot.Add(uint64(i * 32))
//
//				update := &ethpbv2.LightClientUpdate{
//					AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//						Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
//							HeaderAltair: &ethpbv2.LightClientHeader{
//								Beacon: &ethpbv1.BeaconBlockHeader{
//									Slot:          newSlot,
//									ProposerIndex: 1,
//									ParentRoot:    []byte{1, 1, 1},
//									StateRoot:     []byte{1, 1, 1},
//									BodyRoot:      []byte{1, 1, 1},
//								},
//							},
//						},
//					},
//					NextSyncCommittee: &ethpbv2.SyncCommittee{
//						Pubkeys:         nil,
//						AggregatePubkey: nil,
//					},
//					NextSyncCommitteeBranch: nil,
//					FinalizedHeader:         nil,
//					FinalityBranch:          nil,
//					SyncAggregate: &ethpbv1.SyncAggregate{
//						SyncCommitteeBits:      []byte{1, 1, 1},
//						SyncCommitteeSignature: []byte{1, 1, 1},
//					},
//					SignatureSlot: 7,
//				}
//
//				updates[i] = update
//
//				err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//					Version: version.Altair,
//					Data:    update,
//				})
//				require.NoError(t, err)
//
//				updatePeriod++
//			}
//
//			mockChainService := &mock.ChainService{State: st}
//			s := &Server{
//				HeadFetcher: mockChainService,
//				BeaconDB:    db,
//			}
//			startPeriod := 0
//			url := fmt.Sprintf("http://foo.com/?count=10&start_period=%d", startPeriod)
//			request := httptest.NewRequest("GET", url, nil)
//			writer := httptest.NewRecorder()
//			writer.Body = &bytes.Buffer{}
//
//			s.GetLightClientUpdatesByRange(writer, request)
//
//			require.Equal(t, http.StatusOK, writer.Code)
//			var resp structs.LightClientUpdatesByRangeResponse
//			err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//			require.NoError(t, err)
//			require.Equal(t, 1, len(resp.Updates))
//			require.Equal(t, "altair", resp.Updates[0].Version)
//			updateJson, err := structs.LightClientUpdateFromConsensus(updates[0])
//			require.NoError(t, err)
//			require.DeepEqual(t, updateJson, resp.Updates[0].Data)
//		})
//
//		t.Run("missing update at the beginning", func(t *testing.T) {
//			db := setupDB(t)
//
//			updates := make([]*ethpbv2.LightClientUpdate, 3)
//
//			updatePeriod := slot.Div(uint64(config.EpochsPerSyncCommitteePeriod)).Div(uint64(config.SlotsPerEpoch))
//
//			for i := 0; i < 3; i++ {
//				if i == 0 { // skip this update
//					updatePeriod++
//					continue
//				}
//				newSlot := slot.Add(uint64(i))
//
//				update := &ethpbv2.LightClientUpdate{
//					AttestedHeader: &ethpbv2.LightClientHeaderContainer{
//						Header: &ethpbv2.LightClientHeaderContainer_HeaderAltair{
//							HeaderAltair: &ethpbv2.LightClientHeader{
//								Beacon: &ethpbv1.BeaconBlockHeader{
//									Slot:          newSlot,
//									ProposerIndex: 1,
//									ParentRoot:    []byte{1, 1, 1},
//									StateRoot:     []byte{1, 1, 1},
//									BodyRoot:      []byte{1, 1, 1},
//								},
//							},
//						},
//					},
//					NextSyncCommittee: &ethpbv2.SyncCommittee{
//						Pubkeys:         nil,
//						AggregatePubkey: nil,
//					},
//					NextSyncCommitteeBranch: nil,
//					FinalizedHeader:         nil,
//					FinalityBranch:          nil,
//					SyncAggregate: &ethpbv1.SyncAggregate{
//						SyncCommitteeBits:      []byte{1, 1, 1},
//						SyncCommitteeSignature: []byte{1, 1, 1},
//					},
//					SignatureSlot: 7,
//				}
//
//				updates[i] = update
//
//				err = db.SaveLightClientUpdate(ctx, uint64(updatePeriod), &ethpbv2.LightClientUpdateWithVersion{
//					Version: version.Altair,
//					Data:    update,
//				})
//				require.NoError(t, err)
//
//				updatePeriod++
//			}
//
//			mockChainService := &mock.ChainService{State: st}
//			s := &Server{
//				HeadFetcher: mockChainService,
//				BeaconDB:    db,
//			}
//			startPeriod := 0
//			url := fmt.Sprintf("http://foo.com/?count=10&start_period=%d", startPeriod)
//			request := httptest.NewRequest("GET", url, nil)
//			writer := httptest.NewRecorder()
//			writer.Body = &bytes.Buffer{}
//
//			s.GetLightClientUpdatesByRange(writer, request)
//
//			require.Equal(t, http.StatusOK, writer.Code)
//			var resp structs.LightClientUpdatesByRangeResponse
//			err = json.Unmarshal(writer.Body.Bytes(), &resp.Updates)
//			require.NoError(t, err)
//			require.Equal(t, 0, len(resp.Updates))
//		})
//
// }
//
// // TestLightClientHandler_GetLightClientFinalityUpdate tests
//
//	func TestLightClientHandler_GetLightClientFinalityUpdateAltair(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		attestedState, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		err = attestedState.SetSlot(slot.Sub(1))
//		require.NoError(t, err)
//
//		require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
//			Epoch: config.AltairForkEpoch - 10,
//			Root:  make([]byte, 32),
//		}))
//
//		parent := util.NewBeaconBlockAltair()
//		parent.Block.Slot = slot.Sub(1)
//
//		signedParent, err := blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		parentHeader, err := signedParent.Header()
//		require.NoError(t, err)
//		attestedHeader := parentHeader.Header
//
//		err = attestedState.SetLatestBlockHeader(attestedHeader)
//		require.NoError(t, err)
//		attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		parent.Block.StateRoot = attestedStateRoot[:]
//		signedParent, err = blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		err = st.SetSlot(slot)
//		require.NoError(t, err)
//
//		parentRoot, err := signedParent.Block().HashTreeRoot()
//		require.NoError(t, err)
//
//		block := util.NewBeaconBlockAltair()
//		block.Block.Slot = slot
//		block.Block.ParentRoot = parentRoot[:]
//
//		for i := uint64(0); i < config.SyncCommitteeSize; i++ {
//			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
//		}
//
//		signedBlock, err := blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		h, err := signedBlock.Header()
//		require.NoError(t, err)
//
//		err = st.SetLatestBlockHeader(h.Header)
//		require.NoError(t, err)
//		stateRoot, err := st.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		block.Block.StateRoot = stateRoot[:]
//		signedBlock, err = blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		root, err := block.Block.HashTreeRoot()
//		require.NoError(t, err)
//
//		mockBlocker := &testutil.MockBlocker{
//			RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
//				parentRoot: signedParent,
//				root:       signedBlock,
//			},
//			SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
//				slot.Sub(1): signedParent,
//				slot:        signedBlock,
//			},
//		}
//		mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
//			root: true,
//		}}
//		s := &Server{
//			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
//				slot.Sub(1): attestedState,
//				slot:        st,
//			}},
//			Blocker:     mockBlocker,
//			HeadFetcher: mockChainService,
//		}
//		request := httptest.NewRequest("GET", "http://foo.com", nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientFinalityUpdate(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp *structs.LightClientUpdateResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp)
//		require.NoError(t, err)
//		var respHeader structs.LightClientHeader
//		err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
//		require.NoError(t, err)
//		require.Equal(t, "altair", resp.Version)
//		require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
//		require.NotNil(t, resp.Data)
//	}
//
//	func TestLightClientHandler_GetLightClientFinalityUpdateCapella(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		attestedState, err := util.NewBeaconStateCapella()
//		require.NoError(t, err)
//		err = attestedState.SetSlot(slot.Sub(1))
//		require.NoError(t, err)
//
//		require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
//			Epoch: config.AltairForkEpoch - 10,
//			Root:  make([]byte, 32),
//		}))
//
//		parent := util.NewBeaconBlockCapella()
//		parent.Block.Slot = slot.Sub(1)
//
//		signedParent, err := blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		parentHeader, err := signedParent.Header()
//		require.NoError(t, err)
//		attestedHeader := parentHeader.Header
//
//		err = attestedState.SetLatestBlockHeader(attestedHeader)
//		require.NoError(t, err)
//		attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		parent.Block.StateRoot = attestedStateRoot[:]
//		signedParent, err = blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		st, err := util.NewBeaconStateCapella()
//		require.NoError(t, err)
//		err = st.SetSlot(slot)
//		require.NoError(t, err)
//
//		parentRoot, err := signedParent.Block().HashTreeRoot()
//		require.NoError(t, err)
//
//		block := util.NewBeaconBlockCapella()
//		block.Block.Slot = slot
//		block.Block.ParentRoot = parentRoot[:]
//
//		for i := uint64(0); i < config.SyncCommitteeSize; i++ {
//			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
//		}
//
//		signedBlock, err := blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		h, err := signedBlock.Header()
//		require.NoError(t, err)
//
//		err = st.SetLatestBlockHeader(h.Header)
//		require.NoError(t, err)
//		stateRoot, err := st.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		block.Block.StateRoot = stateRoot[:]
//		signedBlock, err = blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		root, err := block.Block.HashTreeRoot()
//		require.NoError(t, err)
//
//		mockBlocker := &testutil.MockBlocker{
//			RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
//				parentRoot: signedParent,
//				root:       signedBlock,
//			},
//			SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
//				slot.Sub(1): signedParent,
//				slot:        signedBlock,
//			},
//		}
//		mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
//			root: true,
//		}}
//		s := &Server{
//			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
//				slot.Sub(1): attestedState,
//				slot:        st,
//			}},
//			Blocker:     mockBlocker,
//			HeadFetcher: mockChainService,
//		}
//		request := httptest.NewRequest("GET", "http://foo.com", nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientFinalityUpdate(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp *structs.LightClientUpdateResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp)
//		require.NoError(t, err)
//		var respHeader structs.LightClientHeader
//		err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
//		require.NoError(t, err)
//		require.Equal(t, "capella", resp.Version)
//		require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
//		require.NotNil(t, resp.Data)
//	}
//
//	func TestLightClientHandler_GetLightClientFinalityUpdateDeneb(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		slot := primitives.Slot(config.DenebForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		attestedState, err := util.NewBeaconStateDeneb()
//		require.NoError(t, err)
//		err = attestedState.SetSlot(slot.Sub(1))
//		require.NoError(t, err)
//
//		require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
//			Epoch: config.AltairForkEpoch - 10,
//			Root:  make([]byte, 32),
//		}))
//
//		parent := util.NewBeaconBlockDeneb()
//		parent.Block.Slot = slot.Sub(1)
//
//		signedParent, err := blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		parentHeader, err := signedParent.Header()
//		require.NoError(t, err)
//		attestedHeader := parentHeader.Header
//
//		err = attestedState.SetLatestBlockHeader(attestedHeader)
//		require.NoError(t, err)
//		attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		parent.Block.StateRoot = attestedStateRoot[:]
//		signedParent, err = blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		st, err := util.NewBeaconStateDeneb()
//		require.NoError(t, err)
//		err = st.SetSlot(slot)
//		require.NoError(t, err)
//
//		parentRoot, err := signedParent.Block().HashTreeRoot()
//		require.NoError(t, err)
//
//		block := util.NewBeaconBlockDeneb()
//		block.Block.Slot = slot
//		block.Block.ParentRoot = parentRoot[:]
//
//		for i := uint64(0); i < config.SyncCommitteeSize; i++ {
//			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
//		}
//
//		signedBlock, err := blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		h, err := signedBlock.Header()
//		require.NoError(t, err)
//
//		err = st.SetLatestBlockHeader(h.Header)
//		require.NoError(t, err)
//		stateRoot, err := st.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		block.Block.StateRoot = stateRoot[:]
//		signedBlock, err = blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		root, err := block.Block.HashTreeRoot()
//		require.NoError(t, err)
//
//		mockBlocker := &testutil.MockBlocker{
//			RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
//				parentRoot: signedParent,
//				root:       signedBlock,
//			},
//			SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
//				slot.Sub(1): signedParent,
//				slot:        signedBlock,
//			},
//		}
//		mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
//			root: true,
//		}}
//		s := &Server{
//			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
//				slot.Sub(1): attestedState,
//				slot:        st,
//			}},
//			Blocker:     mockBlocker,
//			HeadFetcher: mockChainService,
//		}
//		request := httptest.NewRequest("GET", "http://foo.com", nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientFinalityUpdate(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp *structs.LightClientUpdateResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp)
//		require.NoError(t, err)
//		var respHeader structs.LightClientHeaderDeneb
//		err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
//		require.NoError(t, err)
//		require.Equal(t, "deneb", resp.Version)
//		require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
//		require.NotNil(t, resp.Data)
//	}
//
//	func TestLightClientHandler_GetLightClientOptimisticUpdateAltair(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		slot := primitives.Slot(config.AltairForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		attestedState, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		err = attestedState.SetSlot(slot.Sub(1))
//		require.NoError(t, err)
//
//		require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
//			Epoch: config.AltairForkEpoch - 10,
//			Root:  make([]byte, 32),
//		}))
//
//		parent := util.NewBeaconBlockAltair()
//		parent.Block.Slot = slot.Sub(1)
//
//		signedParent, err := blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		parentHeader, err := signedParent.Header()
//		require.NoError(t, err)
//		attestedHeader := parentHeader.Header
//
//		err = attestedState.SetLatestBlockHeader(attestedHeader)
//		require.NoError(t, err)
//		attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		parent.Block.StateRoot = attestedStateRoot[:]
//		signedParent, err = blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		st, err := util.NewBeaconStateAltair()
//		require.NoError(t, err)
//		err = st.SetSlot(slot)
//		require.NoError(t, err)
//
//		parentRoot, err := signedParent.Block().HashTreeRoot()
//		require.NoError(t, err)
//
//		block := util.NewBeaconBlockAltair()
//		block.Block.Slot = slot
//		block.Block.ParentRoot = parentRoot[:]
//
//		for i := uint64(0); i < config.SyncCommitteeSize; i++ {
//			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
//		}
//
//		signedBlock, err := blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		h, err := signedBlock.Header()
//		require.NoError(t, err)
//
//		err = st.SetLatestBlockHeader(h.Header)
//		require.NoError(t, err)
//		stateRoot, err := st.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		block.Block.StateRoot = stateRoot[:]
//		signedBlock, err = blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		root, err := block.Block.HashTreeRoot()
//		require.NoError(t, err)
//
//		mockBlocker := &testutil.MockBlocker{
//			RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
//				parentRoot: signedParent,
//				root:       signedBlock,
//			},
//			SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
//				slot.Sub(1): signedParent,
//				slot:        signedBlock,
//			},
//		}
//		mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
//			root: true,
//		}}
//		s := &Server{
//			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
//				slot.Sub(1): attestedState,
//				slot:        st,
//			}},
//			Blocker:     mockBlocker,
//			HeadFetcher: mockChainService,
//		}
//		request := httptest.NewRequest("GET", "http://foo.com", nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientOptimisticUpdate(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp *structs.LightClientUpdateResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp)
//		require.NoError(t, err)
//		var respHeader structs.LightClientHeader
//		err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
//		require.NoError(t, err)
//		require.Equal(t, "altair", resp.Version)
//		require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
//		require.NotNil(t, resp.Data)
//	}
//
//	func TestLightClientHandler_GetLightClientOptimisticUpdateCapella(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		attestedState, err := util.NewBeaconStateCapella()
//		require.NoError(t, err)
//		err = attestedState.SetSlot(slot.Sub(1))
//		require.NoError(t, err)
//
//		require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
//			Epoch: config.AltairForkEpoch - 10,
//			Root:  make([]byte, 32),
//		}))
//
//		parent := util.NewBeaconBlockCapella()
//		parent.Block.Slot = slot.Sub(1)
//
//		signedParent, err := blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		parentHeader, err := signedParent.Header()
//		require.NoError(t, err)
//		attestedHeader := parentHeader.Header
//
//		err = attestedState.SetLatestBlockHeader(attestedHeader)
//		require.NoError(t, err)
//		attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		parent.Block.StateRoot = attestedStateRoot[:]
//		signedParent, err = blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		st, err := util.NewBeaconStateCapella()
//		require.NoError(t, err)
//		err = st.SetSlot(slot)
//		require.NoError(t, err)
//
//		parentRoot, err := signedParent.Block().HashTreeRoot()
//		require.NoError(t, err)
//
//		block := util.NewBeaconBlockCapella()
//		block.Block.Slot = slot
//		block.Block.ParentRoot = parentRoot[:]
//
//		for i := uint64(0); i < config.SyncCommitteeSize; i++ {
//			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
//		}
//
//		signedBlock, err := blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		h, err := signedBlock.Header()
//		require.NoError(t, err)
//
//		err = st.SetLatestBlockHeader(h.Header)
//		require.NoError(t, err)
//		stateRoot, err := st.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		block.Block.StateRoot = stateRoot[:]
//		signedBlock, err = blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		root, err := block.Block.HashTreeRoot()
//		require.NoError(t, err)
//
//		mockBlocker := &testutil.MockBlocker{
//			RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
//				parentRoot: signedParent,
//				root:       signedBlock,
//			},
//			SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
//				slot.Sub(1): signedParent,
//				slot:        signedBlock,
//			},
//		}
//		mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
//			root: true,
//		}}
//		s := &Server{
//			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
//				slot.Sub(1): attestedState,
//				slot:        st,
//			}},
//			Blocker:     mockBlocker,
//			HeadFetcher: mockChainService,
//		}
//		request := httptest.NewRequest("GET", "http://foo.com", nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientOptimisticUpdate(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp *structs.LightClientUpdateResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp)
//		require.NoError(t, err)
//		var respHeader structs.LightClientHeaderCapella
//		err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
//		require.NoError(t, err)
//		require.Equal(t, "capella", resp.Version)
//		require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
//		require.NotNil(t, resp.Data)
//	}
//
//	func TestLightClientHandler_GetLightClientOptimisticUpdateDeneb(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		slot := primitives.Slot(config.DenebForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		attestedState, err := util.NewBeaconStateDeneb()
//		require.NoError(t, err)
//		err = attestedState.SetSlot(slot.Sub(1))
//		require.NoError(t, err)
//
//		require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
//			Epoch: config.AltairForkEpoch - 10,
//			Root:  make([]byte, 32),
//		}))
//
//		parent := util.NewBeaconBlockDeneb()
//		parent.Block.Slot = slot.Sub(1)
//
//		signedParent, err := blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		parentHeader, err := signedParent.Header()
//		require.NoError(t, err)
//		attestedHeader := parentHeader.Header
//
//		err = attestedState.SetLatestBlockHeader(attestedHeader)
//		require.NoError(t, err)
//		attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		parent.Block.StateRoot = attestedStateRoot[:]
//		signedParent, err = blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		st, err := util.NewBeaconStateDeneb()
//		require.NoError(t, err)
//		err = st.SetSlot(slot)
//		require.NoError(t, err)
//
//		parentRoot, err := signedParent.Block().HashTreeRoot()
//		require.NoError(t, err)
//
//		block := util.NewBeaconBlockDeneb()
//		block.Block.Slot = slot
//		block.Block.ParentRoot = parentRoot[:]
//
//		for i := uint64(0); i < config.SyncCommitteeSize; i++ {
//			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
//		}
//
//		signedBlock, err := blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		h, err := signedBlock.Header()
//		require.NoError(t, err)
//
//		err = st.SetLatestBlockHeader(h.Header)
//		require.NoError(t, err)
//		stateRoot, err := st.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		block.Block.StateRoot = stateRoot[:]
//		signedBlock, err = blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		root, err := block.Block.HashTreeRoot()
//		require.NoError(t, err)
//
//		mockBlocker := &testutil.MockBlocker{
//			RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
//				parentRoot: signedParent,
//				root:       signedBlock,
//			},
//			SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
//				slot.Sub(1): signedParent,
//				slot:        signedBlock,
//			},
//		}
//		mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
//			root: true,
//		}}
//		s := &Server{
//			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
//				slot.Sub(1): attestedState,
//				slot:        st,
//			}},
//			Blocker:     mockBlocker,
//			HeadFetcher: mockChainService,
//		}
//		request := httptest.NewRequest("GET", "http://foo.com", nil)
//		writer := httptest.NewRecorder()
//		writer.Body = &bytes.Buffer{}
//
//		s.GetLightClientOptimisticUpdate(writer, request)
//
//		require.Equal(t, http.StatusOK, writer.Code)
//		var resp *structs.LightClientUpdateResponse
//		err = json.Unmarshal(writer.Body.Bytes(), &resp)
//		require.NoError(t, err)
//		var respHeader structs.LightClientHeaderDeneb
//		err = json.Unmarshal(resp.Data.AttestedHeader, &respHeader)
//		require.NoError(t, err)
//		require.Equal(t, "deneb", resp.Version)
//		require.Equal(t, hexutil.Encode(attestedHeader.BodyRoot), respHeader.Beacon.BodyRoot)
//		require.NotNil(t, resp.Data)
//	}
//
//	func TestLightClientHandler_GetLightClientEventBlock(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		attestedState, err := util.NewBeaconStateCapella()
//		require.NoError(t, err)
//		err = attestedState.SetSlot(slot.Sub(1))
//		require.NoError(t, err)
//
//		require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
//			Epoch: config.AltairForkEpoch - 10,
//			Root:  make([]byte, 32),
//		}))
//
//		parent := util.NewBeaconBlockCapella()
//		parent.Block.Slot = slot.Sub(1)
//
//		signedParent, err := blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		parentHeader, err := signedParent.Header()
//		require.NoError(t, err)
//		attestedHeader := parentHeader.Header
//
//		err = attestedState.SetLatestBlockHeader(attestedHeader)
//		require.NoError(t, err)
//		attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		parent.Block.StateRoot = attestedStateRoot[:]
//		signedParent, err = blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		st, err := util.NewBeaconStateCapella()
//		require.NoError(t, err)
//		err = st.SetSlot(slot)
//		require.NoError(t, err)
//
//		parentRoot, err := signedParent.Block().HashTreeRoot()
//		require.NoError(t, err)
//
//		block := util.NewBeaconBlockCapella()
//		block.Block.Slot = slot
//		block.Block.ParentRoot = parentRoot[:]
//
//		for i := uint64(0); i < config.SyncCommitteeSize; i++ {
//			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
//		}
//
//		signedBlock, err := blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		h, err := signedBlock.Header()
//		require.NoError(t, err)
//
//		err = st.SetLatestBlockHeader(h.Header)
//		require.NoError(t, err)
//		stateRoot, err := st.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		block.Block.StateRoot = stateRoot[:]
//		signedBlock, err = blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		root, err := block.Block.HashTreeRoot()
//		require.NoError(t, err)
//
//		mockBlocker := &testutil.MockBlocker{
//			RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
//				parentRoot: signedParent,
//				root:       signedBlock,
//			},
//			SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
//				slot.Sub(1): signedParent,
//				slot:        signedBlock,
//			},
//		}
//		mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
//			root: true,
//		}}
//		s := &Server{
//			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
//				slot.Sub(1): attestedState,
//				slot:        st,
//			}},
//			Blocker:     mockBlocker,
//			HeadFetcher: mockChainService,
//		}
//
//		minSignaturesRequired := uint64(100)
//		eventBlock, err := s.suitableBlock(ctx, minSignaturesRequired)
//
//		require.NoError(t, err)
//		require.NotNil(t, eventBlock)
//		require.Equal(t, slot, eventBlock.Block().Slot())
//		syncAggregate, err := eventBlock.Block().Body().SyncAggregate()
//		require.NoError(t, err)
//		require.Equal(t, true, syncAggregate.SyncCommitteeBits.Count() >= minSignaturesRequired)
//	}
//
//	func TestLightClientHandler_GetLightClientEventBlock_NeedFetchParent(t *testing.T) {
//		helpers.ClearCache()
//		ctx := context.Background()
//		params.SetupTestConfigCleanup(t)
//		config := params.BeaconConfig()
//		slot := primitives.Slot(config.CapellaForkEpoch * primitives.Epoch(config.SlotsPerEpoch)).Add(1)
//
//		attestedState, err := util.NewBeaconStateCapella()
//		require.NoError(t, err)
//		err = attestedState.SetSlot(slot.Sub(1))
//		require.NoError(t, err)
//
//		require.NoError(t, attestedState.SetFinalizedCheckpoint(&ethpb.Checkpoint{
//			Epoch: config.AltairForkEpoch - 10,
//			Root:  make([]byte, 32),
//		}))
//
//		parent := util.NewBeaconBlockCapella()
//		parent.Block.Slot = slot.Sub(1)
//		for i := uint64(0); i < config.SyncCommitteeSize; i++ {
//			parent.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
//		}
//
//		signedParent, err := blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		parentHeader, err := signedParent.Header()
//		require.NoError(t, err)
//		attestedHeader := parentHeader.Header
//
//		err = attestedState.SetLatestBlockHeader(attestedHeader)
//		require.NoError(t, err)
//		attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		parent.Block.StateRoot = attestedStateRoot[:]
//		signedParent, err = blocks.NewSignedBeaconBlock(parent)
//		require.NoError(t, err)
//
//		st, err := util.NewBeaconStateCapella()
//		require.NoError(t, err)
//		err = st.SetSlot(slot)
//		require.NoError(t, err)
//
//		parentRoot, err := signedParent.Block().HashTreeRoot()
//		require.NoError(t, err)
//
//		block := util.NewBeaconBlockCapella()
//		block.Block.Slot = slot
//		block.Block.ParentRoot = parentRoot[:]
//
//		for i := uint64(0); i < 10; i++ {
//			block.Block.Body.SyncAggregate.SyncCommitteeBits.SetBitAt(i, true)
//		}
//
//		signedBlock, err := blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		h, err := signedBlock.Header()
//		require.NoError(t, err)
//
//		err = st.SetLatestBlockHeader(h.Header)
//		require.NoError(t, err)
//		stateRoot, err := st.HashTreeRoot(ctx)
//		require.NoError(t, err)
//
//		// get a new signed block so the root is updated with the new state root
//		block.Block.StateRoot = stateRoot[:]
//		signedBlock, err = blocks.NewSignedBeaconBlock(block)
//		require.NoError(t, err)
//
//		root, err := block.Block.HashTreeRoot()
//		require.NoError(t, err)
//
//		mockBlocker := &testutil.MockBlocker{
//			RootBlockMap: map[[32]byte]interfaces.ReadOnlySignedBeaconBlock{
//				parentRoot: signedParent,
//				root:       signedBlock,
//			},
//			SlotBlockMap: map[primitives.Slot]interfaces.ReadOnlySignedBeaconBlock{
//				slot.Sub(1): signedParent,
//				slot:        signedBlock,
//			},
//		}
//		mockChainService := &mock.ChainService{Optimistic: true, Slot: &slot, State: st, FinalizedRoots: map[[32]byte]bool{
//			root: true,
//		}}
//		s := &Server{
//			Stater: &testutil.MockStater{StatesBySlot: map[primitives.Slot]state.BeaconState{
//				slot.Sub(1): attestedState,
//				slot:        st,
//			}},
//			Blocker:     mockBlocker,
//			HeadFetcher: mockChainService,
//		}
//
//		minSignaturesRequired := uint64(100)
//		eventBlock, err := s.suitableBlock(ctx, minSignaturesRequired)
//
//		require.NoError(t, err)
//		require.NotNil(t, eventBlock)
//		syncAggregate, err := eventBlock.Block().Body().SyncAggregate()
//		require.NoError(t, err)
//		require.Equal(t, true, syncAggregate.SyncCommitteeBits.Count() >= minSignaturesRequired)
//		require.Equal(t, slot-1, eventBlock.Block().Slot())
//	}
//
// setupDB instantiates and returns a Store instance.
func setupDB(t testing.TB) *kv.Store {
	db, err := kv.NewKVStore(context.Background(), t.TempDir())
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, db.Close(), "Failed to close database")
	})
	return db
}
