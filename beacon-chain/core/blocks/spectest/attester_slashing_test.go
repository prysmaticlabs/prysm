package spectest

import (
	"path"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runAttesterSlashingTest(t *testing.T, config string) {
	if err := spectest.SetConfig(config); err != nil {
		t.Fatal(err)
	}

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "operations/attester_slashing/pyspec_tests")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			attSlashingFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "attester_slashing.ssz")
			if err != nil {
				t.Fatal(err)
			}
			attSlashing := &ethpb.AttesterSlashing{}
			if err := ssz.Unmarshal(attSlashingFile, attSlashing); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			preStatePath := path.Join(testsFolderPath, folder.Name(), "pre.ssz")
			postStatePath := path.Join(testsFolderPath, folder.Name(), "post.ssz")
			body := &ethpb.BeaconBlockBody{AttesterSlashings: []*ethpb.AttesterSlashing{attSlashing}}
			testutil.RunBlockOperationTest(t, preStatePath, body, postStatePath, blocks.ProcessAttesterSlashings)
		})
	}
}
