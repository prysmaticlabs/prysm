package spectest

import (
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

func runAttesterSlashingTest(t *testing.T, config string) {
	folderPath := path.Join("tests", config, "phase0/operations/attester_slashing/pyspec_tests")
	filepath, err := bazel.Runfile(attFolderPath)
	if err != nil {
		t.Fatal(err)
	}
	folders, err := ioutil.ReadDir(filepath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	for _, folder := range folders {
		slashingFilepath, err := bazel.Runfile(path.Join(folderPath, folder.Name(), "attester_slashing.ssz"))
		if err != nil {
			t.Fatal(err)
		}
		slashingFile, err := ioutil.ReadFile(slashingFilepath)
		if err != nil {
			t.Fatal(err)
		}
		attSlashing := &ethpb.AttesterSlashing{}
		if err := ssz.Unmarshal(slashingFile, att); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		file, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatalf("Could not load file %v", err)
		}

		if err := spectest.SetConfig(config); err != nil {
			t.Fatal(err)
		}

		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearAllCaches()

			body := &ethpb.BeaconBlockBody{AttesterSlashings: []*ethpb.AttesterSlashing{attSlashing}}

			postState, err := blocks.ProcessAttesterSlashings(tt.Pre, body)
			// Note: This doesn't test anything worthwhile. It essentially tests
			// that *any* error has occurred, not any specific error.
			if tt.Post == nil {
				if err == nil {
					t.Fatal("Did not fail when expected")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if !proto.Equal(postState, tt.Post) {
				diff, _ := messagediff.PrettyDiff(postState, tt.Post)
				t.Log(diff)
				t.Fatal("Post state does not match expected")
			}
		})
	}
}
