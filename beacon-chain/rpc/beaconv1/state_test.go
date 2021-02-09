package beaconv1

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	powMock "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetGenesis(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.GenesisForkVersion = []byte("genesis")
	params.OverrideBeaconConfig(config)
	genesis := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	validatorsRoot := [32]byte{1, 2, 3, 4, 5, 6}

	t.Run("OK", func(t *testing.T) {
		chainService := &chainMock.ChainService{
			Genesis:        genesis,
			ValidatorsRoot: validatorsRoot,
		}
		s := Server{
			GenesisTimeFetcher: chainService,
			ChainInfoFetcher:   chainService,
		}
		resp, err := s.GetGenesis(ctx, &ptypes.Empty{})
		require.NoError(t, err)
		assert.Equal(t, genesis.Unix(), resp.Data.GenesisTime.Seconds)
		assert.Equal(t, int32(0), resp.Data.GenesisTime.Nanos)
		assert.DeepEqual(t, validatorsRoot[:], resp.Data.GenesisValidatorsRoot)
		assert.DeepEqual(t, []byte("genesis"), resp.Data.GenesisForkVersion)
	})

	t.Run("No genesis time", func(t *testing.T) {
		chainService := &chainMock.ChainService{
			Genesis:        time.Time{},
			ValidatorsRoot: validatorsRoot,
		}
		s := Server{
			GenesisTimeFetcher: chainService,
			ChainInfoFetcher:   chainService,
		}
		_, err := s.GetGenesis(ctx, &ptypes.Empty{})
		assert.ErrorContains(t, "Chain genesis info is not yet known", err)
	})

	t.Run("No genesis validator root", func(t *testing.T) {
		chainService := &chainMock.ChainService{
			Genesis:        genesis,
			ValidatorsRoot: [32]byte{},
		}
		s := Server{
			GenesisTimeFetcher: chainService,
			ChainInfoFetcher:   chainService,
		}
		_, err := s.GetGenesis(ctx, &ptypes.Empty{})
		assert.ErrorContains(t, "Chain genesis info is not yet known", err)
	})
}

func TestGetStateRoot(t *testing.T) {
	ctx := context.Background()
	root := []byte("123456")

	t.Run("Head", func(t *testing.T) {
		chainService := &chainMock.ChainService{
			Root: root,
		}
		s := Server{
			ChainInfoFetcher: chainService,
		}
		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, root, resp.Data.StateRoot)
	})

	t.Run("Genesis", func(t *testing.T) {
		state, err := testutil.NewBeaconState()
		require.NoError(t, err)
		chainFetcher := &powMock.POWChain{
			GenesisState: state,
		}
		s := Server{
			ChainStartFetcher: chainFetcher,
		}
		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("genesis"),
		})
		require.NoError(t, err)
		expectedRoot, err := state.HashTreeRoot(ctx)
		require.NoError(t, err)
		var b [32]byte
		copy(b[:], resp.Data.StateRoot)
		assert.DeepEqual(t, expectedRoot, b)
	})

	t.Run("Finalized", func(t *testing.T) {
		state, err := testutil.NewBeaconState()
		require.NoError(t, err)
		var blockRoot [32]byte
		copy(blockRoot[:], "block_root")
		chainService := &chainMock.ChainService{
			FinalizedCheckPoint: &eth.Checkpoint{
				Root: blockRoot[:],
			},
		}
		stateGen := stategen.NewMockService()
		stateGen.StatesByRoot[blockRoot] = state
		s := Server{
			ChainInfoFetcher: chainService,
			StateGenService:  stateGen,
		}
		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("finalized"),
		})
		require.NoError(t, err)
		expectedRoot, err := state.HashTreeRoot(ctx)
		require.NoError(t, err)
		var b [32]byte
		copy(b[:], resp.Data.StateRoot)
		assert.DeepEqual(t, expectedRoot, b)
	})

	t.Run("Justified", func(t *testing.T) {
		state, err := testutil.NewBeaconState()
		require.NoError(t, err)
		var blockRoot [32]byte
		copy(blockRoot[:], "block_root")
		chainService := &chainMock.ChainService{
			CurrentJustifiedCheckPoint: &eth.Checkpoint{
				Root: blockRoot[:],
			},
		}
		stateGen := stategen.NewMockService()
		stateGen.StatesByRoot[blockRoot] = state
		s := Server{
			ChainInfoFetcher: chainService,
			StateGenService:  stateGen,
		}
		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("justified"),
		})
		require.NoError(t, err)
		expectedRoot, err := state.HashTreeRoot(ctx)
		require.NoError(t, err)
		var b [32]byte
		copy(b[:], resp.Data.StateRoot)
		assert.DeepEqual(t, expectedRoot, b)
	})

	t.Run("Hex root", func(t *testing.T) {
		// We fill state roots with hex representations of natural numbers starting with 1.
		// Example: 16 becomes 0x00...0f
		fillStateRoots := func(state *pb.BeaconState) {
			rootsLen := params.MainnetConfig().SlotsPerHistoricalRoot
			roots := make([][]byte, rootsLen)
			for i := uint64(0); i < rootsLen; i++ {
				roots[i] = make([]byte, 32)
			}
			for j := 0; j < len(roots); j++ {
				// Remove '0x' prefix and left-pad '0' to have 64 chars in total.
				s := fmt.Sprintf("%064s", hexutil.EncodeUint64(uint64(j))[2:])
				h, err := hexutil.Decode("0x" + s)
				require.NoError(t, err, "Failed to decode root "+s)
				roots[j] = h
			}
			state.StateRoots = roots
		}

		state, err := testutil.NewBeaconState(fillStateRoots)
		require.NoError(t, err)
		chainService := &chainMock.ChainService{
			State: state,
		}
		s := Server{
			ChainInfoFetcher: chainService,
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("0", 63) + "1")
		require.NoError(t, err)
		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: stateId,
		})
		require.NoError(t, err)
		assert.DeepEqual(t, stateId, resp.Data.StateRoot)
	})

	t.Run("Hex root not found", func(t *testing.T) {
		state, err := testutil.NewBeaconState()
		require.NoError(t, err)
		chainService := &chainMock.ChainService{
			State: state,
		}
		s := Server{
			ChainInfoFetcher: chainService,
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("f", 64))
		require.NoError(t, err)
		_, err = s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: stateId,
		})
		assert.ErrorContains(t, fmt.Sprintf("State not found in the last %d states", len(state.StateRoots())), err)
	})

	t.Run("Slot", func(t *testing.T) {
		state, err := testutil.NewBeaconState(func(state *pb.BeaconState) {
			state.Slot = 100
		})
		require.NoError(t, err)
		chainService := &chainMock.ChainService{
			State: state,
		}
		stateGen := stategen.NewMockService()
		stateGen.StatesBySlot[100] = state
		s := Server{
			ChainInfoFetcher: chainService,
			StateGenService:  stateGen,
		}
		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("100"),
		})
		require.NoError(t, err)
		expectedRoot, err := state.HashTreeRoot(ctx)
		require.NoError(t, err)
		var b [32]byte
		copy(b[:], resp.Data.StateRoot)
		assert.DeepEqual(t, expectedRoot, b)
	})

	t.Run("Slot too big", func(t *testing.T) {
		state, err := testutil.NewBeaconState()
		require.NoError(t, err)
		chainService := &chainMock.ChainService{
			State: state,
		}
		s := Server{
			ChainInfoFetcher: chainService,
		}
		_, err = s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte(strconv.FormatUint(state.Slot()+1, 10)),
		})
		assert.ErrorContains(t, "Slot cannot be in the future", err)
	})

	t.Run("Invalid state", func(t *testing.T) {
		s := Server{}
		_, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("foo"),
		})
		require.ErrorContains(t, "Invalid state ID: foo", err)
	})
}
