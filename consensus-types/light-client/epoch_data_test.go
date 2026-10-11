package light_client

import (
	"testing"

	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	pb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

// Fork epochs used by every test in this file. Each fork starts 10 epochs
// after the previous one so a window can straddle a boundary.
const (
	testBellatrixEpoch = 10
	testFuluEpoch      = 50
	testGloasEpoch     = 60
)

func setupEpochDataForks(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.AltairForkEpoch = 0
	cfg.BellatrixForkEpoch = testBellatrixEpoch
	cfg.CapellaForkEpoch = 20
	cfg.DenebForkEpoch = 30
	cfg.ElectraForkEpoch = 40
	cfg.FuluForkEpoch = testFuluEpoch
	cfg.GloasForkEpoch = testGloasEpoch
	params.OverrideBeaconConfig(cfg)
}

// blockDataWithLastBlockAt returns a full window where only entry i is a block.
// i < 0 returns an all-empty window.
func blockDataWithLastBlockAt(i int) []*pb.LightClientBlockData {
	bd := make([]*pb.LightClientBlockData, fieldparams.SlotsPerEpoch)
	for j := range bd {
		bd[j] = &pb.LightClientBlockData{StateRoot: make([]byte, 32)}
	}
	if i >= 0 {
		bd[i].StateRoot[0] = 1
	}
	return bd
}

func gloasBlockDataWithLastBlockAt(i int) []*pb.LightClientBlockDataGloas {
	bd := make([]*pb.LightClientBlockDataGloas, fieldparams.SlotsPerEpoch)
	for j := range bd {
		bd[j] = &pb.LightClientBlockDataGloas{StateRoot: make([]byte, 32)}
	}
	if i >= 0 {
		bd[i].StateRoot[0] = 1
	}
	return bd
}

func TestEpochDataVersion(t *testing.T) {
	setupEpochDataForks(t)
	last := fieldparams.SlotsPerEpoch - 1

	tests := []struct {
		name string
		data EpochData
		want int
	}{
		{
			// The window ends on the Gloas fork slot and that slot has a block.
			name: "window straddles Fulu to Gloas, boundary slot has a block",
			data: &pb.LightClientEpochDataGloas{Epoch: testGloasEpoch, BlockData: gloasBlockDataWithLastBlockAt(last)},
			want: version.Gloas,
		},
		{
			// Same window, but the boundary slot is empty, so the last block is
			// still in the Fulu epoch. The epoch alone would say Gloas.
			name: "window straddles Fulu to Gloas, boundary slot empty",
			data: &pb.LightClientEpochDataElectra{Epoch: testGloasEpoch, BlockData: blockDataWithLastBlockAt(last - 1)},
			want: version.Fulu,
		},
		{
			name: "all slots empty falls back to the epoch start slot",
			data: &pb.LightClientEpochDataGloas{Epoch: testGloasEpoch, BlockData: gloasBlockDataWithLastBlockAt(-1)},
			want: version.Gloas,
		},
		{
			// Bellatrix shares the Altair proto type. The version must still be
			// the exact fork, because the fork key and fork digest differ.
			name: "Bellatrix is reported even though it shares the Altair type",
			data: &pb.LightClientEpochDataAltair{Epoch: testBellatrixEpoch, BlockData: blockDataWithLastBlockAt(last)},
			want: version.Bellatrix,
		},
		{
			// Epoch 0 only has the genesis slot inside its window. Walking back
			// from the start slot must not underflow.
			name: "epoch 0 with the genesis block",
			data: &pb.LightClientEpochDataAltair{Epoch: 0, BlockData: blockDataWithLastBlockAt(last)},
			want: version.Altair,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EpochDataVersion(tc.data)
			require.NoError(t, err)
			require.Equal(t, version.String(tc.want), version.String(got))
		})
	}
}

func TestEpochDataVersion_Errors(t *testing.T) {
	setupEpochDataForks(t)

	t.Run("wrong block_data length", func(t *testing.T) {
		d := &pb.LightClientEpochDataAltair{Epoch: 1, BlockData: blockDataWithLastBlockAt(3)[:4]}
		_, err := EpochDataVersion(d)
		require.ErrorContains(t, "block_data entries", err)
	})
	t.Run("block before genesis", func(t *testing.T) {
		d := &pb.LightClientEpochDataAltair{Epoch: 0, BlockData: blockDataWithLastBlockAt(0)}
		_, err := EpochDataVersion(d)
		require.ErrorContains(t, "before genesis", err)
	})
	t.Run("epoch too large", func(t *testing.T) {
		d := &pb.LightClientEpochDataAltair{Epoch: primitives.Epoch(1<<64 - 1), BlockData: blockDataWithLastBlockAt(-1)}
		_, err := EpochDataVersion(d)
		require.NotNil(t, err)
	})
}
