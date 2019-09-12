package spectest

import (
	"bytes"
	"encoding/hex"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestAggregateSignaturesYaml(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/aggregate_sigs/small")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			test := &AggregateSigsTest{}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			var sigs []*bls.Signature
			for _, s := range test.Input {
				sigBytes, err := hex.DecodeString(s[2:])
				if err != nil {
					t.Fatalf("Cannot decode string to bytes: %v", err)
				}
				sig, err := bls.SignatureFromBytes(sigBytes)
				if err != nil {
					t.Fatalf("Unable to unmarshal signature from bytes: %v", err)
				}
				sigs = append(sigs, sig)
			}
			sig := bls.AggregateSignatures(sigs)

			outputBytes, err := hex.DecodeString(test.Output[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			if !bytes.Equal(outputBytes, sig.Marshal()) {
				t.Fatal("Output does not equal marshaled aggregated sig bytes")
			}
		})
	}
}
