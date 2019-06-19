package spectest

import (
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/prysmaticlabs/prysm/shared/params/spectest"
)

func TestShuffleYaml(t *testing.T) {
	yamlDir := "./"
	ext := ".yaml"
	files, err := ioutil.ReadDir(yamlDir)
	if err != nil {
		t.Fatalf("could not read YAML tests directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		extension := string(file.Name()[len(file.Name())-5:])
		if err != nil || extension != ext {
			continue
		}
		filePath := path.Join(yamlDir, file.Name())
		// #nosec G304
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			t.Fatalf("could not read YAML tests directory: %v", err)
		}
		shuffleTest := &ShuffleTest{}
		if err := yaml.Unmarshal(data, shuffleTest); err != nil {
			t.Fatalf("could not unmarshal YAML file into test struct: %v", err)
		}
		if shuffleTest.Config == "minimal" {
			if err := spectest.SetConfig("minimal"); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := spectest.SetConfig("mainnet"); err != nil {
				t.Fatal(err)
			}
		}

		for _, testCase := range shuffleTest.TestCases {
			if err := runShuffleTest(testCase); err != nil {
				t.Fatalf("shuffle test failed: %v", err)
			}
		}

	}
}

// RunShuffleTest uses validator set specified from a YAML file, runs the validator shuffle
// algorithm, then compare the output with the expected output from the YAML file.
func runShuffleTest(testCase *ShuffleTestCase) error {
	seed := common.HexToHash(testCase.Seed)
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
