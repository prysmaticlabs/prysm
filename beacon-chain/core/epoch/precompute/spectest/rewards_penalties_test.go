package spectest

import (
	"context"
	"path"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

type Delta struct {
	Rewards   []uint64 `json:"rewards"`
	Penalties []uint64 `json:"penalties"`
}

var deltaFiles = []string{"source_deltas.yaml", "target_deltas.yaml", "head_deltas.yaml", "inactivity_penalty_deltas.yaml", "inclusion_delay_deltas.yaml"}

func runPrecomputeRewardsAndPenaltiesTests(t *testing.T, config string) {
	require.NoError(t, spectest.SetConfig(t, config))
	testPath := "rewards/core/pyspec_tests"
	testFolders, testsFolderPath := testutil.TestFolders(t, config, testPath)
	for _, folder := range testFolders {
		helpers.ClearCache()
		t.Run(folder.Name(), func(t *testing.T) {
			folderPath := path.Join(testsFolderPath, folder.Name())
			runPrecomputeRewardsAndPenaltiesTest(t, folderPath)
		})
	}
}

func runPrecomputeRewardsAndPenaltiesTest(t *testing.T, testFolderPath string) {
	ctx := context.Background()
	preBeaconStateFile, err := testutil.BazelFileBytes(path.Join(testFolderPath, "pre.ssz"))
	require.NoError(t, err)
	preBeaconStateBase := &pb.BeaconState{}
	require.NoError(t, ssz.Unmarshal(preBeaconStateFile, preBeaconStateBase), "Failed to unmarshal")
	preBeaconState, err := beaconstate.InitializeFromProto(preBeaconStateBase)
	require.NoError(t, err)

	vp, bp, err := precompute.New(ctx, preBeaconState)
	require.NoError(t, err)
	vp, bp, err = precompute.ProcessAttestations(ctx, preBeaconState, vp, bp)
	require.NoError(t, err)

	rewards, penalties, err := precompute.AttestationsDelta(preBeaconState, bp, vp)
	require.NoError(t, err)
	pRewards, err := precompute.ProposersDelta(preBeaconState, bp, vp)
	require.NoError(t, err)
	if len(rewards) != len(penalties) && len(pRewards) != len(pRewards) {
		t.Fatal("Incorrect lengths")
	}
	for i, reward := range rewards {
		rewards[i] = reward + pRewards[i]
	}

	totalSpecTestRewards := make([]uint64, len(rewards))
	totalSpecTestPenalties := make([]uint64, len(penalties))

	for _, dFile := range deltaFiles {
		sourceFile, err := testutil.BazelFileBytes(path.Join(testFolderPath, dFile))
		require.NoError(t, err)
		d := &Delta{}
		err = yaml.Unmarshal(sourceFile, &d)
		if err != nil {
			panic(err)
		}
		for i, reward := range d.Rewards {
			totalSpecTestRewards[i] += reward
		}
		for i, penalty := range d.Penalties {
			totalSpecTestPenalties[i] += penalty
		}
	}

	if !reflect.DeepEqual(rewards, totalSpecTestRewards) {
		t.Error("Rewards don't match")
		t.Log(rewards)
		t.Log(totalSpecTestRewards)
	}
	if !reflect.DeepEqual(penalties, totalSpecTestPenalties) {
		t.Error("Penalties don't match")
		t.Log(penalties)
		t.Log(totalSpecTestPenalties)
	}
}
