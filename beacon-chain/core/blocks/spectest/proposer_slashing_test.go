package spectest

import (
	"path"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func runProposerSlashingTest(t *testing.T, config string) {
	if err := spectest.SetConfig(config); err != nil {
		t.Fatal(err)
	}

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "phase0/operations/proposer_slashing")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			proposerSlashingFile, err := testutil.SSZFileBytes(testsFolderPath, folder.Name(), "proposer_slashing.ssz")
			if err != nil {
				t.Fatal(err)
			}
			proposerSlashing := &ethpb.ProposerSlashing{}
			if err := ssz.Unmarshal(proposerSlashingFile, proposerSlashing); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			preBeaconStateFile, err := testutil.SSZFileBytes(testsFolderPath, folder.Name(), "pre.ssz")
			if err != nil {
				t.Fatal(err)
			}
			preBeaconState := &pb.BeaconState{}
			if err := ssz.Unmarshal(preBeaconStateFile, preBeaconState); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			postStatePath := path.Join(testsFolderPath, folder.Name(), "post.ssz")
			body := &ethpb.BeaconBlockBody{ProposerSlashings: []*ethpb.ProposerSlashing{proposerSlashing}}
			testutil.RunBlockOperationTest(t, preBeaconState, body, postStatePath, blocks.ProcessProposerSlashings)
		})
	}
}
