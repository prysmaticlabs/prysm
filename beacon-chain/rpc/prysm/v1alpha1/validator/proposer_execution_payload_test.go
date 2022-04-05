package validator

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	types "github.com/prysmaticlabs/eth2-types"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	powtesting "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func Test_tDStringToUint256(t *testing.T) {
	i, err := tDStringToUint256("0x0")
	require.NoError(t, err)
	require.DeepEqual(t, uint256.NewInt(0), i)

	i, err = tDStringToUint256("0x10000")
	require.NoError(t, err)
	require.DeepEqual(t, uint256.NewInt(65536), i)

	_, err = tDStringToUint256("100")
	require.ErrorContains(t, "hex string without 0x prefix", err)

	_, err = tDStringToUint256("0xzzzzzz")
	require.ErrorContains(t, "invalid hex string", err)

	_, err = tDStringToUint256("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF" +
		"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	require.ErrorContains(t, "hex number > 256 bits", err)
}

func TestServer_activationEpochNotReached(t *testing.T) {
	require.Equal(t, false, activationEpochNotReached(0))

	cfg := params.BeaconConfig()
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
			cfg := params.BeaconConfig()
			cfg.TerminalBlockHash = tt.terminalBlockHash
			cfg.TerminalBlockHashActivationEpoch = tt.activationEpoch
			params.OverrideBeaconConfig(cfg)

			vs := &Server{
				ExecutionEngineCaller: &powtesting.EngineClient{PayloadIDBytes: tt.payloadID, ErrForkchoiceUpdated: tt.forkchoiceErr},
				HeadFetcher:           &chainMock.ChainService{State: tt.st},
				BeaconDB:              beaconDB,
			}
			_, err := vs.getExecutionPayload(context.Background(), tt.st.Slot(), tt.validatorIndx)
			if tt.errString != "" {
				require.ErrorContains(t, tt.errString, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServer_getPowBlockHashAtTerminalTotalDifficulty(t *testing.T) {
	tests := []struct {
		name                  string
		paramsTd              string
		currentPowBlock       *pb.ExecutionBlock
		parentPowBlock        *pb.ExecutionBlock
		errLatestExecutionBlk error
		wantTerminalBlockHash []byte
		wantExists            bool
		errString             string
	}{
		{
			name:      "config td overflows",
			paramsTd:  "1115792089237316195423570985008687907853269984665640564039457584007913129638912",
			errString: "could not convert terminal total difficulty to uint256",
		},
		{
			name:                  "could not get latest execution block",
			paramsTd:              "1",
			errLatestExecutionBlk: errors.New("blah"),
			errString:             "could not get latest execution block",
		},
		{
			name:      "nil latest execution block",
			paramsTd:  "1",
			errString: "latest execution block is nil",
		},
		{
			name:     "current execution block invalid TD",
			paramsTd: "1",
			currentPowBlock: &pb.ExecutionBlock{
				Hash:            []byte{'a'},
				TotalDifficulty: "1115792089237316195423570985008687907853269984665640564039457584007913129638912",
			},
			errString: "could not convert total difficulty to uint256",
		},
		{
			name:     "current execution block has zero hash parent",
			paramsTd: "2",
			currentPowBlock: &pb.ExecutionBlock{
				Hash:            []byte{'a'},
				ParentHash:      params.BeaconConfig().ZeroHash[:],
				TotalDifficulty: "0x3",
			},
		},
		{
			name:     "could not get parent block",
			paramsTd: "2",
			currentPowBlock: &pb.ExecutionBlock{
				Hash:            []byte{'a'},
				ParentHash:      []byte{'b'},
				TotalDifficulty: "0x3",
			},
			errString: "could not get parent execution block",
		},
		{
			name:     "parent execution block invalid TD",
			paramsTd: "2",
			currentPowBlock: &pb.ExecutionBlock{
				Hash:            []byte{'a'},
				ParentHash:      []byte{'b'},
				TotalDifficulty: "0x3",
			},
			parentPowBlock: &pb.ExecutionBlock{
				Hash:            []byte{'b'},
				ParentHash:      []byte{'c'},
				TotalDifficulty: "1",
			},
			errString: "could not convert total difficulty to uint256",
		},
		{
			name:     "happy case",
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
		{
			name:     "ttd not reached",
			paramsTd: "3",
			currentPowBlock: &pb.ExecutionBlock{
				Hash:            []byte{'a'},
				ParentHash:      []byte{'b'},
				TotalDifficulty: "0x2",
			},
			parentPowBlock: &pb.ExecutionBlock{
				Hash:            []byte{'b'},
				ParentHash:      []byte{'c'},
				TotalDifficulty: "0x1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := params.BeaconConfig()
			cfg.TerminalTotalDifficulty = tt.paramsTd
			params.OverrideBeaconConfig(cfg)
			var m map[[32]byte]*pb.ExecutionBlock
			if tt.parentPowBlock != nil {
				m = map[[32]byte]*pb.ExecutionBlock{
					bytesutil.ToBytes32(tt.parentPowBlock.Hash): tt.parentPowBlock,
				}
			}
			vs := &Server{
				ExecutionEngineCaller: &powtesting.EngineClient{
					ErrLatestExecBlock: tt.errLatestExecutionBlk,
					ExecutionBlock:     tt.currentPowBlock,
					BlockByHashMap:     m,
				},
			}
			b, e, err := vs.getPowBlockHashAtTerminalTotalDifficulty(context.Background())
			if tt.errString != "" {
				require.ErrorContains(t, tt.errString, err)
			} else {
				require.NoError(t, err)
				require.DeepEqual(t, tt.wantExists, e)
				require.DeepEqual(t, tt.wantTerminalBlockHash, b)
			}
		})
	}
}

func TestServer_getTerminalBlockHashIfExists(t *testing.T) {
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
			cfg := params.BeaconConfig()
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
