package spectest

import (
	"path"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func runDepositTest(t *testing.T, config string) {
	testFolders, testsFolderPath := TestFolders(t, config, "deposit")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			depositFile, err := SSZFileBytes(testsFolderPath, folder.Name(), "deposit.ssz")
			if err != nil {
				t.Fatal(err)
			}
			deposit := &ethpb.Deposit{}
			if err := ssz.Unmarshal(depositFile, deposit); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			preBeaconStateFile, err := SSZFileBytes(testsFolderPath, folder.Name(), "pre.ssz")
			if err != nil {
				t.Fatal(err)
			}
			preBeaconState := &pb.BeaconState{}
			if err := ssz.Unmarshal(preBeaconStateFile, preBeaconState); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			postStatePath := path.Join(testsFolderPath, folder.Name(), "post.ssz")
			body := &ethpb.BeaconBlockBody{Deposits: []*ethpb.Deposit{deposit}}
			RunBlockOperationTest(t, preBeaconState, body, postStatePath, blocks.ProcessDeposits)
		})
	}
}
