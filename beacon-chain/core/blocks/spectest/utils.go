package spectest

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
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
	"gopkg.in/d4l3k/messagediff.v1"
)

// operation types take an int and return a string value.
type operation func(*pb.BeaconState, *ethpb.BeaconBlockBody) (*pb.BeaconState, error)

// TestFolders sets the proper config and returns the result of ReadDir
// on the passed in eth2-spec-tests directory along with its path.
func TestFolders(t *testing.T, config string, operation string) ([]os.FileInfo, string) {
	testsFolderPath := path.Join("tests", config, "phase0", operation, "/pyspec_tests")
	filepath, err := bazel.Runfile(testsFolderPath)
	if err != nil {
		t.Fatal(err)
	}
	testFolders, err := ioutil.ReadDir(filepath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if err := spectest.SetConfig(config); err != nil {
		t.Fatal(err)
	}

	return testFolders, testsFolderPath
}

// SSZFileBytes returns the unmarshalled SSZ interface at the passed in path.
func SSZFileBytes(folderPath string, testName string, filename string) ([]byte, error) {
	filepath, err := bazel.Runfile(path.Join(folderPath, testName, filename))
	if err != nil {
		return nil, err
	}
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return fileBytes, nil
}

func RunBlockOperationTest(
	t *testing.T,
	preState *pb.BeaconState,
	body *ethpb.BeaconBlockBody,
	postStatePath string,
	operationFn operation,
) {
	helpers.ClearAllCaches()

	// If the post.ssz is not present, it means the test should fail on our end.
	postSSZFilepath, err := bazel.Runfile(postStatePath)
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
