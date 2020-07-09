package spectest

import (
	"bytes"
	"encoding/hex"
	"path"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bls/iface"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func TestAggregateYaml(t *testing.T) {
	flags := &featureconfig.Flags{}
	reset := featureconfig.InitWithReset(flags)
	t.Run("herumi", testAggregateYaml)
	reset()

	flags.EnableBlst = true
	reset = featureconfig.InitWithReset(flags)
	t.Run("blst", testAggregateYaml)
	reset()
}

func testAggregateYaml(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/aggregate/small")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			test := &AggregateTest{}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			var sigs []iface.Signature
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
			if len(test.Input) == 0 {
				if test.Output != "" {
					t.Fatalf("Output Aggregate is not of zero length:Output %s", test.Output)
				}
				return
			}
			sig := bls.AggregateSignatures(sigs)
			if strings.Contains(folder.Name(), "aggregate_na_pubkeys") {
				if sig != nil {
					t.Errorf("Expected nil signature, received: %v", sig)
				}
				return
			}
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
