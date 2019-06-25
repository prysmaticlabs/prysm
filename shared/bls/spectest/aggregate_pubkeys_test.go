package spectest

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestAggregatePubkeysYaml(t *testing.T) {
	file, err := loadBlsYaml("aggregate_pubkeys/aggregate_pubkeys.yaml")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &AggregatePubkeysTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	for i, tt := range test.TestCases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			pk, err := bls.PublicKeyFromBytes(tt.Input[0])
			if err != nil {
				t.Fatal(err)
			}
			for _, pk2 := range tt.Input[1:] {
				p, err := bls.PublicKeyFromBytes(pk2)
				if err != nil {
					t.Fatal(err)
				}
				pk.Aggregate(p)
			}

			if !bytes.Equal(tt.Output, pk.Marshal()) {
				t.Fatal("Output does not equal marshalled aggregated public " +
					"key bytes")
			}
		})
	}
}
