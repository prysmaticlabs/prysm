package testutil

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	"github.com/gogo/protobuf/proto"
	jsoniter "github.com/json-iterator/go"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"gopkg.in/d4l3k/messagediff.v1"
)

type blockOperation func(context.Context, *beaconstate.BeaconState, *ethpb.BeaconBlockBody) (*beaconstate.BeaconState, error)
type epochOperation func(*testing.T, *beaconstate.BeaconState) (*beaconstate.BeaconState, error)

var json = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            true,
	ValidateJsonRawMessage: true,
	TagKey:                 "spec-name",
}.Froze()

// UnmarshalYaml using a customized json encoder that supports "spec-name"
// override tag.
func UnmarshalYaml(y []byte, dest interface{}) error {
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, dest)
}

// TestFolders sets the proper config and returns the result of ReadDir
// on the passed in eth2-spec-tests directory along with its path.
func TestFolders(t testing.TB, config string, folderPath string) ([]os.FileInfo, string) {
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

// BazelDirectoryNonEmpty returns true if directory exists and is not empty.
func BazelDirectoryNonEmpty(filePath string) (bool, error) {
	p, err := bazel.Runfile(filePath)
	if err != nil {
		return false, err
	}
	fs, err := ioutil.ReadDir(p)
	if err != nil {
		return false, err
	}
	return len(fs) > 0, nil
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
	preBeaconStateFile, err := BazelFileBytes(path.Join(folderPath, "pre.ssz"))
	if err != nil {
		t.Fatal(err)
	}
	preStateBase := &pb.BeaconState{}
	if err := preStateBase.UnmarshalSSZ(preBeaconStateFile); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	preState, err := beaconstate.InitializeFromProto(preStateBase)
	if err != nil {
		t.Fatal(err)
	}

	// If the post.ssz is not present, it means the test should fail on our end.
	postSSZFilepath, err := bazel.Runfile(path.Join(folderPath, "post.ssz"))
	postSSZExists := true
	if err != nil && strings.Contains(err.Error(), "could not locate file") {
		postSSZExists = false
	} else if err != nil {
		t.Fatal(err)
	}

	helpers.ClearCache()
	beaconState, err := operationFn(context.Background(), preState, body)
	if postSSZExists {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		postBeaconStateFile, err := ioutil.ReadFile(postSSZFilepath)
		if err != nil {
			t.Fatal(err)
		}

		postBeaconState := &pb.BeaconState{}
		if err := postBeaconState.UnmarshalSSZ(postBeaconStateFile); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if !proto.Equal(beaconState.InnerStateUnsafe(), postBeaconState) {
			diff, _ := messagediff.PrettyDiff(beaconState.InnerStateUnsafe(), postBeaconState)
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
	preBeaconStateFile, err := BazelFileBytes(path.Join(testFolderPath, "pre.ssz"))
	if err != nil {
		t.Fatal(err)
	}
	preBeaconStateBase := &pb.BeaconState{}
	if err := preBeaconStateBase.UnmarshalSSZ(preBeaconStateFile); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	preBeaconState, err := beaconstate.InitializeFromProto(preBeaconStateBase)
	if err != nil {
		t.Fatal(err)
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
		if err := postBeaconState.UnmarshalSSZ(postBeaconStateFile); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if !proto.Equal(beaconState.InnerStateUnsafe(), postBeaconState) {
			diff, _ := messagediff.PrettyDiff(beaconState.InnerStateUnsafe(), postBeaconState)
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
