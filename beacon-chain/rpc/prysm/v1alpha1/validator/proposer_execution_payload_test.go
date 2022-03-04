package validator

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1/mocks"
	powtesting "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
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
				ExecutionEngineCaller: &mocks.EngineClient{
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
				ExecutionEngineCaller: &mocks.EngineClient{
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
