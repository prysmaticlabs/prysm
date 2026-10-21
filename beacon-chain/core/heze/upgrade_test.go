package heze_test

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/heze"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/time"
	state_native "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v7/config/params"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/testing/util"
)

func TestUpgradeToHeze(t *testing.T) {
	t.Run("upgrades a Gloas state", func(t *testing.T) {
		pre, err := util.NewBeaconStateGloas(func(s *ethpb.BeaconStateGloas) error {
			s.Slot = 64
			s.Fork.CurrentVersion = params.BeaconConfig().GloasForkVersion
			s.Eth1DepositIndex = 7
			s.LatestExecutionPayloadBid.BlockHash[0] = 0xaa
			s.LatestExecutionPayloadBid.Value = 9
			s.LatestBlockHash[0] = 0xbb
			return nil
		})
		require.NoError(t, err)
		container, err := state_native.ContainerFrom[*ethpb.BeaconStateGloas](pre.ToContainer())
		require.NoError(t, err)

		post, err := heze.UpgradeToHeze(pre)
		require.NoError(t, err)
		require.Equal(t, version.Heze, post.Version())

		got, err := state_native.ContainerFrom[*ethpb.BeaconStateHeze](post.ToContainer())
		require.NoError(t, err)
		require.DeepEqual(t, &ethpb.Fork{
			PreviousVersion: params.BeaconConfig().GloasForkVersion,
			CurrentVersion:  params.BeaconConfig().HezeForkVersion,
			Epoch:           time.CurrentEpoch(pre),
		}, got.Fork)
		require.Equal(t, container.Slot, got.Slot)
		require.Equal(t, container.Eth1DepositIndex, got.Eth1DepositIndex)
		require.DeepEqual(t, container.Eth1Data, got.Eth1Data)
		require.DeepEqual(t, container.LatestBlockHash, got.LatestBlockHash)
		require.DeepEqual(t, container.PtcWindow, got.PtcWindow)

		require.DeepEqual(t, container.LatestExecutionPayloadBid, got.LatestExecutionPayloadBid)

		// The upgraded state must hash like the generated SSZ of the same Heze state.
		wantRoot, err := got.HashTreeRoot()
		require.NoError(t, err)
		gotRoot, err := post.HashTreeRoot(t.Context())
		require.NoError(t, err)
		require.Equal(t, wantRoot, gotRoot)
	})

}
