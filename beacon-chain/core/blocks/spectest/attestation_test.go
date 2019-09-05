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

func runAttestationTest(t *testing.T, config string) {
	if err := spectest.SetConfig(config); err != nil {
		t.Fatal(err)
	}

	testFolders, testsFolderPath := testutil.TestFolders(t, config, "operations/attestation/pyspec_tests")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			attestationFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "attestation.ssz")
			if err != nil {
				t.Fatal(err)
			}
			att := &ethpb.Attestation{}
			if err := ssz.Unmarshal(attestationFile, att); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			preBeaconStateFile, err := testutil.BazelFileBytes(testsFolderPath, folder.Name(), "pre.ssz")
			if err != nil {
				t.Fatal(err)
			}
			preBeaconState := &pb.BeaconState{}
			if err := ssz.Unmarshal(preBeaconStateFile, preBeaconState); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			postStatePath := path.Join(testsFolderPath, folder.Name(), "post.ssz")
			body := &ethpb.BeaconBlockBody{Attestations: []*ethpb.Attestation{att}}
			testutil.RunBlockOperationTest(t, preBeaconState, body, postStatePath, blocks.ProcessAttestations)
		})
	}
}
