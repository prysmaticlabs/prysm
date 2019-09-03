package spectest

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"gopkg.in/d4l3k/messagediff.v1"
)

func runAttestationTest(t *testing.T, config string) {
	attFolderPath := path.Join("tests", config, "phase0/operations/attestation/pyspec_tests")
	filepath, err := bazel.Runfile(attFolderPath)
	if err != nil {
		t.Fatal(err)
	}
	folders, err := ioutil.ReadDir(filepath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	for _, folder := range folders {
		innerPath := path.Join(attFolderPath, folder.Name())
		filepath, err = bazel.Runfile(innerPath)
		if err != nil {
			t.Fatal(err)
		}
		files, err := ioutil.ReadDir(filepath)
		if err != nil {
			t.Fatal(err)
		}
		for _, ff := range files {
			t.Log(ff.Name())
		}
		preSSZFilepath, err := bazel.Runfile(path.Join(attFolderPath, folder.Name(), "pre.ssz"))
		if err != nil {
			t.Fatal(err)
		}
		preBeaconStateFile, err := ioutil.ReadFile(preSSZFilepath)
		if err != nil {
			t.Fatal(err)
		}
		attFilepath, err := bazel.Runfile(path.Join(attFolderPath, folder.Name(), "attestation.ssz"))
		if err != nil {
			t.Fatal(err)
		}
		attestationFile, err := ioutil.ReadFile(attFilepath)
		if err != nil {
			t.Fatal(err)
		}
		beaconState := &pb.BeaconState{}
		if err := ssz.Unmarshal(preBeaconStateFile, beaconState); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		att := &ethpb.Attestation{}
		if err := ssz.Unmarshal(attestationFile, att); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if err := spectest.SetConfig(config); err != nil {
			t.Fatal(err)
		}

		t.Run(folder.Name(), func(t *testing.T) {
			helpers.ClearAllCaches()
			body := &ethpb.BeaconBlockBody{
				Attestations: []*ethpb.Attestation{att},
			}

			post, err := blocks.ProcessAttestations(beaconState, body)
			// if !reflect.ValueOf(tt.Post).IsValid() {
			// 	// Note: This doesn't test anything worthwhile. It essentially tests
			// 	// that *any* error has occurred, not any specific error.
			// 	if err == nil {
			// 		t.Fatal("did not fail when expected")
			// 	}
			// 	return
			// }

			// If the post.ssz is not present, it means the transition should fail on our end.
			postBeaconStateFile, err := ioutil.ReadFile(filepath + "/" + folder.Name() + "/post.ssz")
			if err != nil {
				t.Fatal(err)
			}

			// Note: This doesn't test anything worthwhile. It essentially tests
			// that *any* error has occurred, not any specific error.
			if postBeaconStateFile == nil {
				if err == nil {
					t.Fatal("Did not fail when expected")
				}
				t.Logf("Expected failure; failure reason = %v", err)
				return
			} else if err != nil {
				t.Fatal(err)
			}

			postBeaconState := &pb.BeaconState{}
			if err := ssz.Unmarshal(postBeaconStateFile, postBeaconState); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if !proto.Equal(post, postBeaconState) {
				diff, _ := messagediff.PrettyDiff(post, postBeaconState)
				t.Log(diff)
				t.Fatal("Post state does not match expected")
			}
		})
	}
}
