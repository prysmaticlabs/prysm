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
)

type Delta struct {
	Rewards   []uint64 `json:"rewards"`
	Penalties []uint64 `json:"penalties"`
}

var deltaFiles = []string{"source_deltas.yaml", "target_deltas.yaml", "head_deltas.yaml", "inactivity_penalty_deltas.yaml", "inclusion_delay_deltas.yaml"}

func runPrecomputeRewardsAndPenaltiesTests(t *testing.T, config string) {
	if err := spectest.SetConfig(t, config); err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	preBeaconStateBase := &pb.BeaconState{}
	if err := ssz.Unmarshal(preBeaconStateFile, preBeaconStateBase); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	preBeaconState, err := beaconstate.InitializeFromProto(preBeaconStateBase)
	if err != nil {
		t.Fatal(err)
	}

	vp, bp, err := precompute.New(ctx, preBeaconState)
	if err != nil {
		t.Fatal(err)
	}
	vp, bp, err = precompute.ProcessAttestations(ctx, preBeaconState, vp, bp)
	if err != nil {
		t.Fatal(err)
	}

	rewards, penalties, err := precompute.AttestationsDelta(preBeaconState, bp, vp)
	if err != nil {
		t.Fatal(err)
	}
	pRewards, err := precompute.ProposersDelta(preBeaconState, bp, vp)
	if err != nil {
		t.Fatal(err)
	}
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
		if err != nil {
			t.Fatal(err)
		}
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
