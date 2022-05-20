package forkchoice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	forkchoicetypes "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/time/slots"
)

type Builder struct {
	service  *blockchain.Service
	lastTick int64
	execMock *engineMock
}

func NewBuilder(t testing.TB, initialState state.BeaconState, initialBlock interfaces.SignedBeaconBlock) *Builder {
	execMock := &engineMock{
		powBlocks: make(map[[32]byte]*ethpb.PowBlock),
	}
	service := startChainService(t, initialState, initialBlock, execMock)
	return &Builder{
		service:  service,
		execMock: execMock,
	}
}

// Tick resets the genesis time to now()-tick and adjusts the slot to the appropriate value.
func (bb *Builder) Tick(t testing.TB, tick int64) {
	bb.service.SetGenesisTime(time.Unix(time.Now().Unix()-tick, 0))
	if tick > bb.lastTick {
		slot := uint64(tick) / params.BeaconConfig().SecondsPerSlot
		require.NoError(t, bb.service.NewSlot(context.TODO(), types.Slot(slot)))
		bb.lastTick = tick
	}
}

// block provides the block to forkchoice proposer boost and returns the block root.
func (bb *Builder) block(t testing.TB, b interfaces.SignedBeaconBlock) [32]byte {
	ctx := context.TODO()
	r, err := b.Block().HashTreeRoot()
	require.NoError(t, err)
	slotsSinceGenesis := slots.SinceGenesis(bb.service.GenesisTime())
	args := &forkchoicetypes.ProposerBoostRootArgs{
		BlockRoot:       r,
		BlockSlot:       b.Block().Slot(),
		CurrentSlot:     slotsSinceGenesis,
		SecondsIntoSlot: uint64(bb.lastTick) % params.BeaconConfig().SecondsPerSlot,
	}
	require.NoError(t, bb.service.ForkChoicer().BoostProposerRoot(ctx, args))
	return r
}

// InvalidBlock receives the invalid block and notifies forkchoice.
func (bb *Builder) InvalidBlock(t testing.TB, b interfaces.SignedBeaconBlock) {
	r := bb.block(t, b)
	require.Equal(t, true, bb.service.ReceiveBlock(context.TODO(), b, r) != nil)
}

// ValidBlock receives the valid block and notifies forkchoice.
func (bb *Builder) ValidBlock(t testing.TB, b interfaces.SignedBeaconBlock) {
	r := bb.block(t, b)
	require.NoError(t, bb.service.ReceiveBlock(context.TODO(), b, r))
}

// PoWBlock receives the block and notifies a mocked execution engine.
func (bb *Builder) PoWBlock(t testing.TB, pb *ethpb.PowBlock) {
	bb.execMock.powBlocks[bytesutil.ToBytes32(pb.BlockHash)] = pb
}

// Attestation receives the attestation and updates forkchoice.
func (bb *Builder) Attestation(t testing.TB, a *ethpb.Attestation) {
	require.NoError(t, bb.service.OnAttestation(context.TODO(), a))
}

// Check evaluates the fork choice results and compares them to the expected values.
func (bb *Builder) Check(t testing.TB, c *Check) {
	if c == nil {
		return
	}
	ctx := context.TODO()

	require.NoError(t, bb.service.UpdateAndSaveHeadWithBalances(ctx))
	if c.Head != nil {
		r, err := bb.service.HeadRoot(ctx)
		require.NoError(t, err)
		require.DeepEqual(t, common.FromHex(c.Head.Root), r)
		require.Equal(t, types.Slot(c.Head.Slot), bb.service.HeadSlot())
	}
	if c.JustifiedCheckPoint != nil {
		cp := &ethpb.Checkpoint{
			Epoch: types.Epoch(c.JustifiedCheckPoint.Epoch),
			Root:  common.FromHex(c.JustifiedCheckPoint.Root),
		}
		got, err := bb.service.CurrentJustifiedCheckpt()
		require.NoError(t, err)
		require.DeepEqual(t, cp, got)
	}
	if c.BestJustifiedCheckPoint != nil {
		cp := &ethpb.Checkpoint{
			Epoch: types.Epoch(c.BestJustifiedCheckPoint.Epoch),
			Root:  common.FromHex(c.BestJustifiedCheckPoint.Root),
		}
		got, err := bb.service.BestJustifiedCheckpt()
		require.NoError(t, err)
		require.DeepEqual(t, cp, got)
	}
	if c.FinalizedCheckPoint != nil {
		cp := &ethpb.Checkpoint{
			Epoch: types.Epoch(c.FinalizedCheckPoint.Epoch),
			Root:  common.FromHex(c.FinalizedCheckPoint.Root),
		}
		got, err := bb.service.FinalizedCheckpt()
		require.NoError(t, err)
		require.DeepSSZEqual(t, cp, got)
	}
	if c.ProposerBoostRoot != nil {
		want := fmt.Sprintf("%#x", common.FromHex(*c.ProposerBoostRoot))
		got := fmt.Sprintf("%#x", bb.service.ForkChoiceStore().ProposerBoost())
		require.DeepEqual(t, want, got)
	}
}
