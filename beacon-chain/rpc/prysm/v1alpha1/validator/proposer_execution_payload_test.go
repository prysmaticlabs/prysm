package validator

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	powtesting "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestServer_activationEpochNotReached(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	require.Equal(t, false, activationEpochNotReached(0))

	cfg := params.BeaconConfig().Copy()
	cfg.TerminalBlockHash = common.BytesToHash(bytesutil.PadTo([]byte{0x01}, 32))
	cfg.TerminalBlockHashActivationEpoch = 1
	params.OverrideBeaconConfig(cfg)

	require.Equal(t, true, activationEpochNotReached(0))
	require.Equal(t, false, activationEpochNotReached(params.BeaconConfig().SlotsPerEpoch+1))
}

func TestServer_getExecutionPayload(t *testing.T) {
	nonTransitionSt, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	b1pb := util.NewBeaconBlock()
	b1r, err := b1pb.Block.HashTreeRoot()
	require.NoError(t, err)
	b1, err := wrapper.WrappedSignedBeaconBlock(b1pb)
	require.NoError(t, err)
	require.NoError(t, nonTransitionSt.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Root: b1r[:],
	}))

	transitionSt, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	require.NoError(t, transitionSt.SetLatestExecutionPayloadHeader(&ethpb.ExecutionPayloadHeader{BlockNumber: 1}))
	b2pb := util.NewBeaconBlockBellatrix()
	b2r, err := b2pb.Block.HashTreeRoot()
	require.NoError(t, err)
	b2, err := wrapper.WrappedSignedBeaconBlock(b2pb)
	require.NoError(t, err)
	require.NoError(t, transitionSt.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Root: b2r[:],
	}))

	beaconDB := dbTest.SetupDB(t)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), b1))
	require.NoError(t, beaconDB.SaveBlock(context.Background(), b2))
	require.NoError(t, beaconDB.SaveFeeRecipientsByValidatorIDs(context.Background(), []types.ValidatorIndex{0}, []common.Address{{}}))

	tests := []struct {
		name              string
		st                state.BeaconState
		errString         string
		forkchoiceErr     error
		payloadID         *pb.PayloadIDBytes
		terminalBlockHash common.Hash
		activationEpoch   types.Epoch
		validatorIndx     types.ValidatorIndex
	}{
		{
			name:      "transition completed, nil payload id",
			st:        transitionSt,
			errString: "nil payload id",
		},
		{
			name:      "transition completed, happy case (has fee recipient in Db)",
			st:        transitionSt,
			payloadID: &pb.PayloadIDBytes{0x1},
		},
		{
			name:          "transition completed, happy case (doesn't have fee recipient in Db)",
			st:            transitionSt,
			payloadID:     &pb.PayloadIDBytes{0x1},
			validatorIndx: 1,
		},
		{
			name:          "transition completed, happy case, payload ID cached)",
			st:            transitionSt,
			validatorIndx: 100,
		},
		{
			name:          "transition completed, could not prepare payload",
			st:            transitionSt,
			forkchoiceErr: errors.New("fork choice error"),
			errString:     "could not prepare payload",
		},
		{
			name:      "transition not-completed, latest exec block is nil",
			st:        nonTransitionSt,
			errString: "latest execution block is nil",
		},
		{
			name:              "transition not-completed, activation epoch not reached",
			st:                nonTransitionSt,
			terminalBlockHash: [32]byte{0x1},
			activationEpoch:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := params.BeaconConfig().Copy()
			cfg.TerminalBlockHash = tt.terminalBlockHash
			cfg.TerminalBlockHashActivationEpoch = tt.activationEpoch
			params.OverrideBeaconConfig(cfg)

			vs := &Server{
				ExecutionEngineCaller:  &powtesting.EngineClient{PayloadIDBytes: tt.payloadID, ErrForkchoiceUpdated: tt.forkchoiceErr},
				HeadFetcher:            &chainMock.ChainService{State: tt.st},
				BeaconDB:               beaconDB,
				ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
			}
			vs.ProposerSlotIndexCache.SetProposerAndPayloadIDs(tt.st.Slot(), 100, [8]byte{100})
			_, err := vs.getExecutionPayload(context.Background(), tt.st.Slot(), tt.validatorIndx)
			if tt.errString != "" {
				require.ErrorContains(t, tt.errString, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServer_getExecutionPayload_UnexpectedFeeRecipient(t *testing.T) {
	hook := logTest.NewGlobal()
	nonTransitionSt, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	b1pb := util.NewBeaconBlock()
	b1r, err := b1pb.Block.HashTreeRoot()
	require.NoError(t, err)
	b1, err := wrapper.WrappedSignedBeaconBlock(b1pb)
	require.NoError(t, err)
	require.NoError(t, nonTransitionSt.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Root: b1r[:],
	}))

	transitionSt, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	require.NoError(t, transitionSt.SetLatestExecutionPayloadHeader(&ethpb.ExecutionPayloadHeader{BlockNumber: 1}))
	b2pb := util.NewBeaconBlockBellatrix()
	b2r, err := b2pb.Block.HashTreeRoot()
	require.NoError(t, err)
	b2, err := wrapper.WrappedSignedBeaconBlock(b2pb)
	require.NoError(t, err)
	require.NoError(t, transitionSt.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Root: b2r[:],
	}))

	beaconDB := dbTest.SetupDB(t)
	require.NoError(t, beaconDB.SaveBlock(context.Background(), b1))
	require.NoError(t, beaconDB.SaveBlock(context.Background(), b2))
	feeRecipient := common.BytesToAddress([]byte("a"))
	require.NoError(t, beaconDB.SaveFeeRecipientsByValidatorIDs(context.Background(), []types.ValidatorIndex{0}, []common.Address{
		feeRecipient,
	}))

	payloadID := &pb.PayloadIDBytes{0x1}
	payload := emptyPayload()
	payload.FeeRecipient = feeRecipient[:]
	vs := &Server{
		ExecutionEngineCaller: &powtesting.EngineClient{
			PayloadIDBytes:   payloadID,
			ExecutionPayload: payload,
		},
		HeadFetcher:            &chainMock.ChainService{State: transitionSt},
		BeaconDB:               beaconDB,
		ProposerSlotIndexCache: cache.NewProposerPayloadIDsCache(),
	}
	gotPayload, err := vs.getExecutionPayload(context.Background(), transitionSt.Slot(), 0)
	require.NoError(t, err)
	require.NotNil(t, gotPayload)

	// We should NOT be getting the warning.
	require.LogsDoNotContain(t, hook, "Fee recipient address from execution client is not what was expected")
	hook.Reset()

	evilRecipientAddress := common.BytesToAddress([]byte("evil"))
	payload.FeeRecipient = evilRecipientAddress[:]
	vs.ProposerSlotIndexCache = cache.NewProposerPayloadIDsCache()

	gotPayload, err = vs.getExecutionPayload(context.Background(), transitionSt.Slot(), 0)
	require.NoError(t, err)
	require.NotNil(t, gotPayload)

	// Users should be warned.
	require.LogsContain(t, hook, "Fee recipient address from execution client is not what was expected")
}

func TestServer_getTerminalBlockHashIfExists(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	tests := []struct {
		name                  string
		paramsTerminalHash    []byte
		paramsTd              string
		currentPowBlock       *pb.ExecutionBlock
		parentPowBlock        *pb.ExecutionBlock
		wantTerminalBlockHash []byte
		wantExists            bool
		errString             string
	}{
		{
			name:               "use terminal block hash, doesn't exist",
			paramsTerminalHash: []byte{'a'},
			errString:          "could not fetch height for hash",
		},
		{
			name: "use terminal block hash, exists",
			paramsTerminalHash: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			wantExists: true,
			wantTerminalBlockHash: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
		{
			name:     "use terminal total difficulty",
			paramsTd: "2",
			currentPowBlock: &pb.ExecutionBlock{
				Hash:            []byte{'a'},
				ParentHash:      []byte{'b'},
				TotalDifficulty: "0x3",
			},
			parentPowBlock: &pb.ExecutionBlock{
				Hash:            []byte{'b'},
				ParentHash:      []byte{'c'},
				TotalDifficulty: "0x1",
			},
			wantExists:            true,
			wantTerminalBlockHash: []byte{'a'},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := params.BeaconConfig().Copy()
			cfg.TerminalTotalDifficulty = tt.paramsTd
			cfg.TerminalBlockHash = common.BytesToHash(tt.paramsTerminalHash)
			params.OverrideBeaconConfig(cfg)
			var m map[[32]byte]*pb.ExecutionBlock
			if tt.parentPowBlock != nil {
				m = map[[32]byte]*pb.ExecutionBlock{
					bytesutil.ToBytes32(tt.parentPowBlock.Hash): tt.parentPowBlock,
				}
			}
			c := powtesting.NewPOWChain()
			c.HashesByHeight[0] = tt.wantTerminalBlockHash
			vs := &Server{
				Eth1BlockFetcher: c,
				ExecutionEngineCaller: &powtesting.EngineClient{
					ExecutionBlock: tt.currentPowBlock,
					BlockByHashMap: m,
				},
			}
			b, e, err := vs.getTerminalBlockHashIfExists(context.Background())
			if tt.errString != "" {
				require.ErrorContains(t, tt.errString, err)
				require.DeepEqual(t, tt.wantExists, e)
			} else {
				require.NoError(t, err)
				require.DeepEqual(t, tt.wantExists, e)
				require.DeepEqual(t, tt.wantTerminalBlockHash, b)
			}
		})
	}
}
