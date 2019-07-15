package spectest

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

const shufflePrefix = "tests/shuffling/core/"

func TestShufflingMinimal(t *testing.T) {
	helpers.ClearAllCaches()
	filepath, err := bazel.Runfile(shufflePrefix + "shuffling_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runShuffleTests(t, filepath)
}

func TestShufflingMainnet(t *testing.T) {
	helpers.ClearAllCaches()
	filepath, err := bazel.Runfile(shufflePrefix + "shuffling_full.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runShuffleTests(t, filepath)
}

func runShuffleTests(t *testing.T, filepath string) {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("could not read YAML tests directory: %v", err)
	}

	shuffleTest := &ShuffleTest{}
	if err := yaml.Unmarshal(file, shuffleTest); err != nil {
		t.Fatalf("could not unmarshal YAML file into test struct: %v", err)
	}
	if err := spectest.SetConfig(shuffleTest.Config); err != nil {
		t.Fatal(err)
	}
	t.Logf("Title: %v", shuffleTest.Title)
	t.Logf("Summary: %v", shuffleTest.Summary)
	t.Logf("Fork: %v", shuffleTest.Forks)
	t.Logf("Config: %v", shuffleTest.Config)
	for _, testCase := range shuffleTest.TestCases {
		if err := runShuffleTest(testCase); err != nil {
			t.Fatalf("shuffle test failed: %v", err)
		}
	}

}

// RunShuffleTest uses validator set specified from a YAML file, runs the validator shuffle
// algorithm, then compare the output with the expected output from the YAML file.
func runShuffleTest(testCase *ShuffleTestCase) error {
	baseSeed, err := base64.StdEncoding.DecodeString(testCase.Seed)
	if err != nil {
		return err
	}

	seed := common.BytesToHash(baseSeed)
	testIndices := make([]uint64, testCase.Count, testCase.Count)
	for i := uint64(0); i < testCase.Count; i++ {
		testIndices[i] = i
	}
	shuffledList := make([]uint64, testCase.Count)
	for i := uint64(0); i < testCase.Count; i++ {
		si, err := utils.ShuffledIndex(i, testCase.Count, seed)
		if err != nil {
			return err
		}
		shuffledList[i] = si
	}
	if !reflect.DeepEqual(shuffledList, testCase.Shuffled) {
		return fmt.Errorf("shuffle result error: expected %v, actual %v", testCase.Shuffled, shuffledList)
	}
	return nil
}
