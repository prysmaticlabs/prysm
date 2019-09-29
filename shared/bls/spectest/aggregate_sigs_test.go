package spectest

import (
	"bytes"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"

	"gopkg.in/yaml.v2"
)

type aggregateSignaturesTest struct {
	Input  []string `yaml:"input"`
	Output string   `yaml:"output"`
}

func TestAggregateSignatures(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/aggregate_sigs/small")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			test := &aggregateSignaturesTest{}
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatal(err)
			}
			expectedOutputString := toBytes(96, test.Output)
			signatures := []*bls.Signature{}
			for i := 0; i < len(test.Input); i++ {
				signature, err := bls.SignatureFromBytes(toBytes(96, test.Input[i]))
				if err != nil {
					t.Fatal(err)
				}
				signatures = append(signatures, signature)
			}
			aggregated := bls.AggregateSignatures(signatures)
			if !bytes.Equal(expectedOutputString, aggregated.Marshal()) {
				t.Fatal("signature aggregation fails\n", folder.Name())
			}
		})
	}
}
