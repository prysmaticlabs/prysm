package testutil

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"gopkg.in/d4l3k/messagediff.v1"
)

type blockOperation func(*pb.BeaconState, *ethpb.BeaconBlockBody) (*pb.BeaconState, error)
type epochOperation func(*testing.T, *pb.BeaconState) (*pb.BeaconState, error)

// TestFolders sets the proper config and returns the result of ReadDir
// on the passed in eth2-spec-tests directory along with its path.
func TestFolders(t *testing.T, config string, folderPath string) ([]os.FileInfo, string) {
	testsFolderPath := path.Join("tests", config, "phase0", folderPath)
	filepath, err := bazel.Runfile(testsFolderPath)
	if err != nil {
		t.Fatal(err)
	}
	testFolders, err := ioutil.ReadDir(filepath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	return testFolders, testsFolderPath
}

// BazelFileBytes returns the byte array of the bazel file path given.
func BazelFileBytes(filePaths ...string) ([]byte, error) {
	filepath, err := bazel.Runfile(path.Join(filePaths...))
	if err != nil {
		return nil, err
	}
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return fileBytes, nil
}

// RunBlockOperationTest takes in the prestate and the beacon block body, processes it through the
// passed in block operation function and checks the post state with the expected post state.
func RunBlockOperationTest(
	t *testing.T,
	folderPath string,
	body *ethpb.BeaconBlockBody,
	operationFn blockOperation,
) {
	helpers.ClearAllCaches()

	preBeaconStateFile, err := BazelFileBytes(path.Join(folderPath, "pre.ssz"))
	if err != nil {
		t.Fatal(err)
	}
	preState := &pb.BeaconState{}
	if err := ssz.Unmarshal(preBeaconStateFile, preState); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// If the post.ssz is not present, it means the test should fail on our end.
	postSSZFilepath, err := bazel.Runfile(path.Join(folderPath, "post.ssz"))
	postSSZExists := true
	if err != nil && strings.Contains(err.Error(), "could not locate file") {
		postSSZExists = false
	} else if err != nil {
		t.Fatal(err)
	}

	beaconState, err := operationFn(preState, body)
	if postSSZExists {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		postBeaconStateFile, err := ioutil.ReadFile(postSSZFilepath)
		if err != nil {
			t.Fatal(err)
		}

		postBeaconState := &pb.BeaconState{}
		if err := ssz.Unmarshal(postBeaconStateFile, postBeaconState); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if !proto.Equal(beaconState, postBeaconState) {
			diff, _ := messagediff.PrettyDiff(beaconState, postBeaconState)
			t.Log(diff)
			t.Fatal("Post state does not match expected")
		}
	} else {
		// Note: This doesn't test anything worthwhile. It essentially tests
		// that *any* error has occurred, not any specific error.
		if err == nil {
			t.Fatal("Did not fail when expected")
		}
		t.Logf("Expected failure; failure reason = %v", err)
		return
	}
}

// RunEpochOperationTest takes in the prestate and processes it through the
// passed in epoch operation function and checks the post state with the expected post state.
func RunEpochOperationTest(
	t *testing.T,
	testFolderPath string,
	operationFn epochOperation,
) {
	helpers.ClearAllCaches()

	preBeaconStateFile, err := BazelFileBytes(path.Join(testFolderPath, "pre.ssz"))
	if err != nil {
		t.Fatal(err)
	}
	preBeaconState := &pb.BeaconState{}
	if err := ssz.Unmarshal(preBeaconStateFile, preBeaconState); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// If the post.ssz is not present, it means the test should fail on our end.
	postSSZFilepath, err := bazel.Runfile(path.Join(testFolderPath, "post.ssz"))
	postSSZExists := true
	if err != nil && strings.Contains(err.Error(), "could not locate file") {
		postSSZExists = false
	} else if err != nil {
		t.Fatal(err)
	}

	beaconState, err := operationFn(t, preBeaconState)
	if postSSZExists {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		postBeaconStateFile, err := ioutil.ReadFile(postSSZFilepath)
		if err != nil {
			t.Fatal(err)
		}

		postBeaconState := &pb.BeaconState{}
		if err := ssz.Unmarshal(postBeaconStateFile, postBeaconState); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if !proto.Equal(beaconState, postBeaconState) {
			diff, _ := messagediff.PrettyDiff(beaconState, postBeaconState)
			t.Log(diff)
			t.Fatal("Post state does not match expected")
		}
	} else {
		// Note: This doesn't test anything worthwhile. It essentially tests
		// that *any* error has occurred, not any specific error.
		if err == nil {
			t.Fatal("Did not fail when expected")
		}
		t.Logf("Expected failure; failure reason = %v", err)
		return
	}
}
