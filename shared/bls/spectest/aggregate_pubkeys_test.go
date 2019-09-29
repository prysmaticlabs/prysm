package spectest

import (
	"bytes"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"

	"gopkg.in/yaml.v2"
)

type aggregatePubkeysTest struct {
	Input  []string `yaml:"input"`
	Output string   `yaml:"output"`
}

func TestAggregatePubkeys(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/aggregate_pubkeys/small")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			test := &aggregatePubkeysTest{}
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatal(err)
			}
			expectedOutputString := toBytes(48, test.Output)
			aggregated := bls.NewAggregatePubkey()
			for i := 0; i < len(test.Input); i++ {
				pubkey, err := bls.PublicKeyFromBytes(toBytes(48, test.Input[i]))
				if err != nil {
					t.Fatal(err)
				}
				aggregated.Aggregate(pubkey)
			}
			if !bytes.Equal(expectedOutputString, aggregated.Marshal()) {
				t.Fatal("pubkey aggregation fails\n", folder.Name())
			}
		})
	}
}
