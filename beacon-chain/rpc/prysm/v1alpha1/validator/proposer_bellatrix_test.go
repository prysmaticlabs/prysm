package validator

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api/client/builder"
	blockchainTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	builderTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/builder/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	dbTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	powtesting "github.com/prysmaticlabs/prysm/v4/beacon-chain/execution/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz"
	v1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestServer_setExecutionData(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()

	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.BellatrixForkEpoch = 0
	cfg.CapellaForkEpoch = 0
	cfg.InitializeForkSchedule()
	params.OverrideBeaconConfig(cfg)

	beaconDB := dbTest.SetupDB(t)
	capellaTransitionState, _ := util.DeterministicGenesisStateCapella(t, 1)
	denebTransitionState, _ := util.DeterministicGenesisStateDeneb(t, 1)

	withdrawals := []*v1.Withdrawal{{
		Index:          1,
		ValidatorIndex: 2,
		Address:        make([]byte, fieldparams.FeeRecipientLength),
		Amount:         3,
	}}
	id := &v1.PayloadIDBytes{0x1}
	vs := &Server{
		ExecutionEngineCaller:  &powtesting.EngineClient{PayloadIDBytes: id, ExecutionPayloadCapella: &v1.ExecutionPayloadCapella{BlockNumber: 1, Withdrawals: withdrawals}, BlockValue: 0},
		HeadFetcher:            &blockchainTest.ChainService{State: capellaTransitionState},
		FinalizationFetcher:    &blockchainTest.ChainService{},
		BeaconDB:               beaconDB,
		PayloadIDCache:         cache.NewPayloadIDCache(),
		ForkchoiceFetcher:      &blockchainTest.ChainService{},
		TrackedValidatorsCache: cache.NewTrackedValidatorsCache(),
	}

	t.Run("No builder configured. Use local block", func(t *testing.T) {
		vs.BlockBuilder = builderTest.DefaultBuilderService(t, version.Capella, false)

		blk, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockCapella())
		require.NoError(t, err)
		b := blk.Block()
		localPayload, _, err := vs.getLocalPayload(ctx, b, capellaTransitionState)
		require.NoError(t, err)
		builderPayload, builderKzgCommitments, err := vs.getBuilderPayloadAndBlobs(ctx, b.Slot(), b.ProposerIndex())
		require.NoError(t, err)
		require.Equal(t, true, builderPayload == nil)
		require.DeepEqual(t, [][]uint8(nil), builderKzgCommitments)
		require.NoError(t, setExecutionData(context.Background(), blk, localPayload, builderPayload, builderKzgCommitments, defaultBuilderBoostFactor))
		e, err := blk.Block().Body().Execution()
		require.NoError(t, err)
		require.Equal(t, uint64(1), e.BlockNumber()) // Local block
	})
	t.Run("Builder configured. Builder Block has higher value. Incorrect withdrawals", func(t *testing.T) {
		bb := builderTest.DefaultBuilderService(t, version.Capella, true)
		bb.BidCapella.Header.BlockNumber = 2
		bb.Cfg = &builderTest.Config{BeaconDB: beaconDB}
		vs.BlockBuilder = bb

		blk, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockCapella())
		require.NoError(t, err)
		require.NoError(t, vs.BeaconDB.SaveRegistrationsByValidatorIDs(
			ctx,
			[]primitives.ValidatorIndex{blk.Block().ProposerIndex()},
			[]*ethpb.ValidatorRegistrationV1{{FeeRecipient: make([]byte, fieldparams.FeeRecipientLength), Timestamp: uint64(util.DefaultTime.Unix()), Pubkey: make([]byte, fieldparams.BLSPubkeyLength)}}),
		)
		chain := &blockchainTest.ChainService{Genesis: util.DefaultTime, Block: blk}
		vs.TimeFetcher = chain
		vs.HeadFetcher = chain

		b := blk.Block()
		localPayload, _, err := vs.getLocalPayload(ctx, b, capellaTransitionState)
		require.NoError(t, err)
		builderPayload, builderKzgCommitments, err := vs.getBuilderPayloadAndBlobs(ctx, b.Slot(), b.ProposerIndex())
		require.NoError(t, err)
		require.NotNil(t, builderPayload)
		require.DeepEqual(t, [][]uint8(nil), builderKzgCommitments)
		require.NoError(t, setExecutionData(context.Background(), blk, localPayload, builderPayload, builderKzgCommitments, defaultBuilderBoostFactor))
		e, err := blk.Block().Body().Execution()
		require.NoError(t, err)
		require.Equal(t, uint64(1), e.BlockNumber()) // Local block because incorrect withdrawals

		assert.LogsContain(t, hook, "Proposer: withdrawal roots don't match, using local block")
	})
	t.Run("Builder configured. Builder Block has higher value. Correct withdrawals.", func(t *testing.T) {
		bb := builderTest.DefaultBuilderService(t, version.Capella, true)
		wr, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		require.NoError(t, err)
		bb.BidCapella.Header.WithdrawalsRoot = wr[:]
		bb.BidCapella.Header.BlockNumber = 2
		bb.Cfg = &builderTest.Config{BeaconDB: beaconDB}
		vs.BlockBuilder = bb

		blk, err := blocks.NewSignedBeaconBlock(util.NewBlindedBeaconBlockCapella())
		require.NoError(t, err)
		require.NoError(t, vs.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{blk.Block().ProposerIndex()},
			[]*ethpb.ValidatorRegistrationV1{{FeeRecipient: make([]byte, fieldparams.FeeRecipientLength), Timestamp: uint64(util.DefaultTime.Unix()), Pubkey: make([]byte, fieldparams.BLSPubkeyLength)}}))
		chain := &blockchainTest.ChainService{Genesis: util.DefaultTime, Block: blk}
		vs.TimeFetcher = chain
		vs.HeadFetcher = chain

		b := blk.Block()
		localPayload, _, err := vs.getLocalPayload(ctx, b, capellaTransitionState)
		require.NoError(t, err)
		builderPayload, builderKzgCommitments, err := vs.getBuilderPayloadAndBlobs(ctx, b.Slot(), b.ProposerIndex())
		require.NoError(t, err)
		require.NotNil(t, builderPayload)
		require.DeepEqual(t, [][]uint8(nil), builderKzgCommitments)
		require.NoError(t, setExecutionData(context.Background(), blk, localPayload, builderPayload, builderKzgCommitments, defaultBuilderBoostFactor))
		e, err := blk.Block().Body().Execution()
		require.NoError(t, err)
		require.Equal(t, uint64(2), e.BlockNumber()) // Builder block
	})
	t.Run("Max builder boost factor should return builder", func(t *testing.T) {
		bb := builderTest.DefaultBuilderService(t, version.Capella, true)
		wr, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		require.NoError(t, err)
		bb.BidCapella.Header.WithdrawalsRoot = wr[:]
		bb.BidCapella.Header.BlockNumber = 2
		bb.Cfg = &builderTest.Config{BeaconDB: beaconDB}
		vs.BlockBuilder = bb

		blk, err := blocks.NewSignedBeaconBlock(util.NewBlindedBeaconBlockCapella())
		require.NoError(t, err)
		require.NoError(t, vs.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{blk.Block().ProposerIndex()},
			[]*ethpb.ValidatorRegistrationV1{{FeeRecipient: make([]byte, fieldparams.FeeRecipientLength), Timestamp: uint64(util.DefaultTime.Unix()), Pubkey: make([]byte, fieldparams.BLSPubkeyLength)}}))
		chain := &blockchainTest.ChainService{Genesis: util.DefaultTime, Block: blk}
		vs.TimeFetcher = chain
		vs.HeadFetcher = chain

		b := blk.Block()
		localPayload, _, err := vs.getLocalPayload(ctx, b, capellaTransitionState)
		require.NoError(t, err)
		builderPayload, builderKzgCommitments, err := vs.getBuilderPayloadAndBlobs(ctx, b.Slot(), b.ProposerIndex())
		require.NoError(t, err)
		require.NotNil(t, builderPayload)
		require.DeepEqual(t, [][]uint8(nil), builderKzgCommitments)
		require.NoError(t, setExecutionData(context.Background(), blk, localPayload, builderPayload, builderKzgCommitments, math.MaxUint64))
		e, err := blk.Block().Body().Execution()
		require.NoError(t, err)
		require.Equal(t, uint64(2), e.BlockNumber()) // builder block
	})
	t.Run("Builder builder has higher value but forced to local payload with builder boost factor", func(t *testing.T) {
		bb := builderTest.DefaultBuilderService(t, version.Capella, true)
		wr, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		require.NoError(t, err)
		bb.BidCapella.Header.WithdrawalsRoot = wr[:]
		bb.BidCapella.Header.BlockNumber = 2
		bb.Cfg = &builderTest.Config{BeaconDB: beaconDB}
		vs.BlockBuilder = bb

		blk, err := blocks.NewSignedBeaconBlock(util.NewBlindedBeaconBlockCapella())
		require.NoError(t, err)
		require.NoError(t, vs.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{blk.Block().ProposerIndex()},
			[]*ethpb.ValidatorRegistrationV1{{FeeRecipient: make([]byte, fieldparams.FeeRecipientLength), Timestamp: uint64(util.DefaultTime.Unix()), Pubkey: make([]byte, fieldparams.BLSPubkeyLength)}}))
		chain := &blockchainTest.ChainService{Genesis: util.DefaultTime, Block: blk}
		vs.TimeFetcher = chain
		vs.HeadFetcher = chain

		b := blk.Block()
		localPayload, _, err := vs.getLocalPayload(ctx, b, capellaTransitionState)
		require.NoError(t, err)
		builderPayload, builderKzgCommitments, err := vs.getBuilderPayloadAndBlobs(ctx, b.Slot(), b.ProposerIndex())
		require.NoError(t, err)
		require.NotNil(t, builderPayload)
		require.DeepEqual(t, [][]uint8(nil), builderKzgCommitments)
		require.NoError(t, setExecutionData(context.Background(), blk, localPayload, builderPayload, builderKzgCommitments, 0))
		e, err := blk.Block().Body().Execution()
		require.NoError(t, err)
		require.Equal(t, uint64(1), e.BlockNumber()) // local block
	})
	t.Run("Builder configured. Local block has higher value", func(t *testing.T) {
		bb := builderTest.DefaultBuilderService(t, version.Capella, true)
		bb.BidCapella.Header.BlockNumber = 2
		bb.Cfg = &builderTest.Config{BeaconDB: beaconDB}
		vs.BlockBuilder = bb

		blk, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockCapella())
		require.NoError(t, err)
		require.NoError(t, vs.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{blk.Block().ProposerIndex()},
			[]*ethpb.ValidatorRegistrationV1{{FeeRecipient: make([]byte, fieldparams.FeeRecipientLength), Timestamp: uint64(util.DefaultTime.Unix()), Pubkey: make([]byte, fieldparams.BLSPubkeyLength)}}))
		vs.ExecutionEngineCaller = &powtesting.EngineClient{PayloadIDBytes: id, ExecutionPayloadCapella: &v1.ExecutionPayloadCapella{BlockNumber: 3}, BlockValue: 2}
		chain := &blockchainTest.ChainService{Genesis: util.DefaultTime, Block: blk}
		vs.TimeFetcher = chain
		vs.HeadFetcher = chain

		b := blk.Block()
		localPayload, _, err := vs.getLocalPayload(ctx, b, capellaTransitionState)
		require.NoError(t, err)
		builderPayload, builderKzgCommitments, err := vs.getBuilderPayloadAndBlobs(ctx, b.Slot(), b.ProposerIndex())
		require.NoError(t, err)
		require.NotNil(t, builderPayload)
		require.DeepEqual(t, [][]uint8(nil), builderKzgCommitments)
		require.NoError(t, setExecutionData(context.Background(), blk, localPayload, builderPayload, builderKzgCommitments, defaultBuilderBoostFactor))
		e, err := blk.Block().Body().Execution()
		require.NoError(t, err)
		require.Equal(t, uint64(3), e.BlockNumber()) // Local block

		require.LogsContain(t, hook, "builderGweiValue=1 localBoostPercentage=0 localGweiValue=2")
	})
	t.Run("Builder configured. Local block and local boost has higher value", func(t *testing.T) {
		cfg := params.BeaconConfig().Copy()
		cfg.LocalBlockValueBoost = 1 // Boost 1%.
		undo, err := params.SetActiveWithUndo(cfg)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, undo())
		}()

		bb := builderTest.DefaultBuilderService(t, version.Capella, true)
		bb.BidCapella.Header.BlockNumber = 2
		bb.Cfg = &builderTest.Config{BeaconDB: beaconDB}
		vs.BlockBuilder = bb

		blk, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockCapella())
		require.NoError(t, err)
		require.NoError(t, vs.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{blk.Block().ProposerIndex()},
			[]*ethpb.ValidatorRegistrationV1{{FeeRecipient: make([]byte, fieldparams.FeeRecipientLength), Timestamp: uint64(util.DefaultTime.Unix()), Pubkey: make([]byte, fieldparams.BLSPubkeyLength)}}))
		vs.ExecutionEngineCaller = &powtesting.EngineClient{PayloadIDBytes: id, ExecutionPayloadCapella: &v1.ExecutionPayloadCapella{BlockNumber: 3}, BlockValue: 1}
		chain := &blockchainTest.ChainService{Genesis: util.DefaultTime, Block: blk}
		vs.TimeFetcher = chain
		vs.HeadFetcher = chain

		b := blk.Block()
		localPayload, _, err := vs.getLocalPayload(ctx, b, capellaTransitionState)
		require.NoError(t, err)
		builderPayload, builderKzgCommitments, err := vs.getBuilderPayloadAndBlobs(ctx, b.Slot(), b.ProposerIndex())
		require.NoError(t, err)
		require.NotNil(t, builderPayload)
		require.DeepEqual(t, [][]uint8(nil), builderKzgCommitments)
		require.NoError(t, setExecutionData(context.Background(), blk, localPayload, builderPayload, builderKzgCommitments, defaultBuilderBoostFactor))
		e, err := blk.Block().Body().Execution()
		require.NoError(t, err)
		require.Equal(t, uint64(3), e.BlockNumber()) // Local block

		require.LogsContain(t, hook, "builderGweiValue=1 localBoostPercentage=1 localGweiValue=1")
	})
	t.Run("Builder configured. Builder returns fault. Use local block", func(t *testing.T) {
		bb := builderTest.DefaultBuilderService(t, version.Capella, true)
		bb.ErrGetHeader = errors.New("fault")
		bb.Cfg = &builderTest.Config{BeaconDB: beaconDB}
		vs.BlockBuilder = bb

		blk, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockCapella())
		require.NoError(t, err)
		require.NoError(t, vs.BeaconDB.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{blk.Block().ProposerIndex()},
			[]*ethpb.ValidatorRegistrationV1{{FeeRecipient: make([]byte, fieldparams.FeeRecipientLength), Timestamp: uint64(util.DefaultTime.Unix()), Pubkey: make([]byte, fieldparams.BLSPubkeyLength)}}))
		vs.ExecutionEngineCaller = &powtesting.EngineClient{PayloadIDBytes: id, ExecutionPayloadCapella: &v1.ExecutionPayloadCapella{BlockNumber: 4}, BlockValue: 0}
		chain := &blockchainTest.ChainService{Genesis: util.DefaultTime, Block: blk}
		vs.TimeFetcher = chain
		vs.HeadFetcher = chain

		b := blk.Block()
		localPayload, _, err := vs.getLocalPayload(ctx, b, capellaTransitionState)
		require.NoError(t, err)
		builderPayload, builderKzgCommitments, err := vs.getBuilderPayloadAndBlobs(ctx, b.Slot(), b.ProposerIndex())
		require.ErrorContains(t, "fault", err) // Builder returns fault. Use local block
		require.NoError(t, setExecutionData(context.Background(), blk, localPayload, builderPayload, builderKzgCommitments, defaultBuilderBoostFactor))
		e, err := blk.Block().Body().Execution()
		require.NoError(t, err)
		require.Equal(t, uint64(4), e.BlockNumber()) // Local block
	})
	t.Run("Can get local payload and blobs Deneb", func(t *testing.T) {
		cfg := params.BeaconConfig().Copy()
		cfg.DenebForkEpoch = 0
		undo, err := params.SetActiveWithUndo(cfg)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, undo())
		}()

		vs.BlockBuilder = builderTest.DefaultBuilderService(t, version.Deneb, false)
		blobsBundle := &v1.BlobsBundle{
			KzgCommitments: [][]byte{{1, 2, 3}},
			Proofs:         [][]byte{{4, 5, 6}},
			Blobs:          [][]byte{{7, 8, 9}},
		}
		vs.ExecutionEngineCaller = &powtesting.EngineClient{
			PayloadIDBytes:        id,
			BlobsBundle:           blobsBundle,
			ExecutionPayloadDeneb: &v1.ExecutionPayloadDeneb{BlockNumber: 4},
			BlockValue:            0}

		blk, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockDeneb())
		require.NoError(t, err)
		blk.SetSlot(1)
		localPayload, _, err := vs.getLocalPayload(ctx, blk.Block(), capellaTransitionState)
		require.NoError(t, err)
		require.Equal(t, uint64(4), localPayload.BlockNumber())
		cachedBundle := bundleCache.get(blk.Block().Slot())
		require.DeepEqual(t, cachedBundle, blobsBundle)
	})
	t.Run("Can get builder payload and blobs in Deneb", func(t *testing.T) {
		cfg := params.BeaconConfig().Copy()
		cfg.DenebForkEpoch = 0
		undo, err := params.SetActiveWithUndo(cfg)
		require.NoError(t, err)
		defer func() { require.NoError(t, undo()) }()

		kzgCommitments := [][]byte{bytesutil.PadTo([]byte{2}, fieldparams.BLSPubkeyLength), bytesutil.PadTo([]byte{5}, fieldparams.BLSPubkeyLength)}

		bb := builderTest.DefaultBuilderService(t, version.Deneb, true)
		wr, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
		require.NoError(t, err)
		bb.BidDeneb.Header.WithdrawalsRoot = wr[:]
		bb.BidDeneb.Header.BlockNumber = 2
		bb.BidDeneb.BlobKzgCommitments = kzgCommitments
		bb.Cfg = &builderTest.Config{BeaconDB: beaconDB}
		vs.BlockBuilder = bb

		blk, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockDeneb())
		require.NoError(t, err)
		require.NoError(t, beaconDB.SaveRegistrationsByValidatorIDs(ctx, []primitives.ValidatorIndex{blk.Block().ProposerIndex()},
			[]*ethpb.ValidatorRegistrationV1{{FeeRecipient: make([]byte, fieldparams.FeeRecipientLength), Timestamp: uint64(util.DefaultTime.Unix()), Pubkey: make([]byte, fieldparams.BLSPubkeyLength)}}))
		vs.ExecutionEngineCaller = &powtesting.EngineClient{PayloadIDBytes: id, ExecutionPayloadDeneb: &v1.ExecutionPayloadDeneb{BlockNumber: 4, Withdrawals: withdrawals}, BlockValue: 0}
		chain := &blockchainTest.ChainService{Genesis: util.DefaultTime, Block: blk}
		vs.TimeFetcher = chain
		vs.HeadFetcher = chain

		builderPayload, builderKzgCommitments, err := vs.getBuilderPayloadAndBlobs(ctx, blk.Block().Slot(), blk.Block().ProposerIndex())
		require.NoError(t, err)
		require.DeepEqual(t, kzgCommitments, builderKzgCommitments)
		require.Equal(t, uint64(2), builderPayload.BlockNumber()) // header should be the same from block
		localPayload, _, err := vs.getLocalPayload(ctx, blk.Block(), denebTransitionState)
		require.NoError(t, err)
		require.NoError(t, setExecutionData(context.Background(), blk, localPayload, builderPayload, builderKzgCommitments, defaultBuilderBoostFactor))
		got, err := blk.Block().Body().BlobKzgCommitments()
		require.NoError(t, err)
		require.DeepEqual(t, kzgCommitments, got)
	})
}

func TestServer_getPayloadHeader(t *testing.T) {
	genesis := time.Now().Add(-time.Duration(params.BeaconConfig().SlotsPerEpoch) * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)
	params.SetupTestConfigCleanup(t)
	bc := params.BeaconConfig()
	bc.BellatrixForkEpoch = 1
	fakeCapellaEpoch := primitives.Epoch(10)
	bc.CapellaForkEpoch = fakeCapellaEpoch
	bc.InitializeForkSchedule()
	params.OverrideBeaconConfig(bc)

	emptyRoot, err := ssz.TransactionsRoot([][]byte{})
	require.NoError(t, err)
	ti, err := slots.ToTime(uint64(time.Now().Unix()), 0)
	require.NoError(t, err)
	fakeBid, err := util.DefaultBid()
	require.NoError(t, err)
	fakeBid.Header.Timestamp = uint64(ti.Unix())
	tiCapella, err := slots.ToTime(uint64(genesis.Unix()), primitives.Slot(fakeCapellaEpoch)*params.BeaconConfig().SlotsPerEpoch)
	require.NoError(t, err)
	fakeBidCapella, err := util.DefaultBidCapella()
	require.NoError(t, err)
	fakeBidCapella.Header.Timestamp = uint64(tiCapella.Unix())

	tests := []struct {
		name                  string
		head                  interfaces.ReadOnlySignedBeaconBlock
		builderService        func() *builderTest.MockBuilderService
		fetcher               *blockchainTest.ChainService
		err                   string
		returnedHeader        *v1.ExecutionPayloadHeader
		returnedHeaderCapella *v1.ExecutionPayloadHeaderCapella
	}{
		{
			name: "can't request before bellatrix epoch",
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.ReadOnlySignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
					require.NoError(t, err)
					return wb
				}(),
			},
			err: "can't get payload header from builder before bellatrix epoch",
		},
		{
			name: "get header failed",
			builderService: func() *builderTest.MockBuilderService {
				s := builderTest.DefaultBuilderService(t, version.Bellatrix, true)
				s.ErrGetHeader = errors.New("can't get header")
				s.Bid = fakeBid
				return s
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.ReadOnlySignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					wb.SetSlot(primitives.Slot(params.BeaconConfig().BellatrixForkEpoch) * params.BeaconConfig().SlotsPerEpoch)
					return wb
				}(),
			},
			err: "can't get header",
		},
		{
			name: "0 bid",
			builderService: func() *builderTest.MockBuilderService {
				s := builderTest.DefaultBuilderService(t, version.Bellatrix, true)
				s.Bid.Value = bytesutil.PadTo([]byte{}, 32)
				return s
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.ReadOnlySignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					wb.SetSlot(primitives.Slot(params.BeaconConfig().BellatrixForkEpoch) * params.BeaconConfig().SlotsPerEpoch)
					return wb
				}(),
			},
			err: "builder returned header with 0 bid amount",
		},
		{
			name: "invalid tx root",
			builderService: func() *builderTest.MockBuilderService {
				s := builderTest.DefaultBuilderService(t, version.Bellatrix, true)
				s.Bid.Header.TransactionsRoot = emptyRoot[:]
				return s
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.ReadOnlySignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					wb.SetSlot(primitives.Slot(params.BeaconConfig().BellatrixForkEpoch) * params.BeaconConfig().SlotsPerEpoch)
					return wb
				}(),
			},
			err: "builder returned header with an empty tx root",
		},
		{
			name: "can get header",
			builderService: func() *builderTest.MockBuilderService {
				s := builderTest.DefaultBuilderService(t, version.Bellatrix, true)
				s.Bid = fakeBid
				return s
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.ReadOnlySignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					wb.SetSlot(primitives.Slot(params.BeaconConfig().BellatrixForkEpoch) * params.BeaconConfig().SlotsPerEpoch)
					return wb
				}(),
			},
			returnedHeader: fakeBid.Header,
		},
		{
			name: "wrong bid version",
			builderService: func() *builderTest.MockBuilderService {
				return builderTest.DefaultBuilderService(t, version.Capella, true)
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.ReadOnlySignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					wb.SetSlot(primitives.Slot(params.BeaconConfig().BellatrixForkEpoch) * params.BeaconConfig().SlotsPerEpoch)
					return wb
				}(),
			},
			err: "is different from head block version",
		},
		{
			name: "different bid version during hard fork",
			builderService: func() *builderTest.MockBuilderService {
				s := builderTest.DefaultBuilderService(t, version.Capella, true)
				s.BidCapella = fakeBidCapella
				return s
			},
			fetcher: &blockchainTest.ChainService{
				Block: func() interfaces.ReadOnlySignedBeaconBlock {
					wb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
					require.NoError(t, err)
					wb.SetSlot(primitives.Slot(fakeCapellaEpoch) * params.BeaconConfig().SlotsPerEpoch)
					return wb
				}(),
			},
			returnedHeaderCapella: fakeBidCapella.Header,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bb *builderTest.MockBuilderService
			if tc.builderService != nil {
				bb = tc.builderService()
			}
			vs := &Server{BlockBuilder: bb, HeadFetcher: tc.fetcher, TimeFetcher: &blockchainTest.ChainService{
				Genesis: genesis,
			}}
			hb, err := vs.HeadFetcher.HeadBlock(context.Background())
			require.NoError(t, err)
			h, _, err := vs.getPayloadHeaderFromBuilder(context.Background(), hb.Block().Slot(), 0)
			if tc.err != "" {
				require.ErrorContains(t, tc.err, err)
			} else {
				require.NoError(t, err)
				if tc.returnedHeader != nil {
					want, err := blocks.WrappedExecutionPayloadHeader(tc.returnedHeader)
					require.NoError(t, err)
					require.DeepEqual(t, want, h)
				}
				if tc.returnedHeaderCapella != nil {
					want, err := blocks.WrappedExecutionPayloadHeaderCapella(tc.returnedHeaderCapella, blocks.PayloadValueToGwei(bb.BidCapella.Value))
					require.NoError(t, err)
					require.DeepEqual(t, want, h)
				}
			}
		})
	}
}

func TestServer_validateBuilderSignature(t *testing.T) {
	sk, err := bls.RandKey()
	require.NoError(t, err)
	bid := &ethpb.BuilderBid{
		Header: &v1.ExecutionPayloadHeader{
			ParentHash:       make([]byte, fieldparams.RootLength),
			FeeRecipient:     make([]byte, fieldparams.FeeRecipientLength),
			StateRoot:        make([]byte, fieldparams.RootLength),
			ReceiptsRoot:     make([]byte, fieldparams.RootLength),
			LogsBloom:        make([]byte, fieldparams.LogsBloomLength),
			PrevRandao:       make([]byte, fieldparams.RootLength),
			BaseFeePerGas:    make([]byte, fieldparams.RootLength),
			BlockHash:        make([]byte, fieldparams.RootLength),
			TransactionsRoot: make([]byte, fieldparams.RootLength),
			BlockNumber:      1,
		},
		Pubkey: sk.PublicKey().Marshal(),
		Value:  bytesutil.PadTo([]byte{1, 2, 3}, 32),
	}
	d := params.BeaconConfig().DomainApplicationBuilder
	domain, err := signing.ComputeDomain(d, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(bid, domain)
	require.NoError(t, err)
	pbBid := &ethpb.SignedBuilderBid{
		Message:   bid,
		Signature: sk.Sign(sr[:]).Marshal(),
	}
	sBid, err := builder.WrappedSignedBuilderBid(pbBid)
	require.NoError(t, err)
	require.NoError(t, validateBuilderSignature(sBid))

	pbBid.Message.Value = make([]byte, 32)
	sBid, err = builder.WrappedSignedBuilderBid(pbBid)
	require.NoError(t, err)
	require.ErrorIs(t, validateBuilderSignature(sBid), signing.ErrSigFailedToVerify)
}

func Test_matchingWithdrawalsRoot(t *testing.T) {
	t.Run("could not get local withdrawals", func(t *testing.T) {
		local := &v1.ExecutionPayload{}
		p, err := blocks.WrappedExecutionPayload(local)
		require.NoError(t, err)
		_, err = matchingWithdrawalsRoot(p, p)
		require.ErrorContains(t, "could not get local withdrawals", err)
	})
	t.Run("could not get builder withdrawals root", func(t *testing.T) {
		local := &v1.ExecutionPayloadCapella{}
		p, err := blocks.WrappedExecutionPayloadCapella(local, 0)
		require.NoError(t, err)
		header := &v1.ExecutionPayloadHeader{}
		h, err := blocks.WrappedExecutionPayloadHeader(header)
		require.NoError(t, err)
		_, err = matchingWithdrawalsRoot(p, h)
		require.ErrorContains(t, "could not get builder withdrawals root", err)
	})
	t.Run("withdrawals mismatch", func(t *testing.T) {
		local := &v1.ExecutionPayloadCapella{}
		p, err := blocks.WrappedExecutionPayloadCapella(local, 0)
		require.NoError(t, err)
		header := &v1.ExecutionPayloadHeaderCapella{}
		h, err := blocks.WrappedExecutionPayloadHeaderCapella(header, 0)
		require.NoError(t, err)
		matched, err := matchingWithdrawalsRoot(p, h)
		require.NoError(t, err)
		require.Equal(t, false, matched)
	})
	t.Run("withdrawals match", func(t *testing.T) {
		wds := []*v1.Withdrawal{{
			Index:          1,
			ValidatorIndex: 2,
			Address:        make([]byte, fieldparams.FeeRecipientLength),
			Amount:         3,
		}}
		local := &v1.ExecutionPayloadCapella{Withdrawals: wds}
		p, err := blocks.WrappedExecutionPayloadCapella(local, 0)
		require.NoError(t, err)
		header := &v1.ExecutionPayloadHeaderCapella{}
		wr, err := ssz.WithdrawalSliceRoot(wds, fieldparams.MaxWithdrawalsPerPayload)
		require.NoError(t, err)
		header.WithdrawalsRoot = wr[:]
		h, err := blocks.WrappedExecutionPayloadHeaderCapella(header, 0)
		require.NoError(t, err)
		matched, err := matchingWithdrawalsRoot(p, h)
		require.NoError(t, err)
		require.Equal(t, true, matched)
	})
}

func TestEmptyTransactionsRoot(t *testing.T) {
	r, err := ssz.TransactionsRoot([][]byte{})
	require.NoError(t, err)
	require.DeepEqual(t, r, emptyTransactionsRoot)
}
