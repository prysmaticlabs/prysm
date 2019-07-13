package spectest

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"

	"github.com/ghodss/yaml"
)

func TestAggregateSignaturesYaml(t *testing.T) {
	file, err := loadBlsYaml("aggregate_sigs/aggregate_sigs.yaml")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &AggregateSigsTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	for i, tt := range test.TestCases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			var sigs []*bls.Signature
			for _, s := range tt.Input {
				sig, err := bls.SignatureFromBytes(s)
				if err != nil {
					t.Fatalf("Unable to unmarshal signature from bytes: %v", err)
				}
				sigs = append(sigs, sig)
			}
			sig := bls.AggregateSignatures(sigs)
			if !bytes.Equal(tt.Output, sig.Marshal()) {
				t.Fatal("Output does not equal marshaled aggregated sig bytes")
			}
		})
	}
}
