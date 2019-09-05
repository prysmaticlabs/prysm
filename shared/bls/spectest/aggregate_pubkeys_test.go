package spectest

import (
	"bytes"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestAggregatePubkeysYaml(t *testing.T) {
	file, err := loadBlsYaml("aggregate_pubkeys/small/agg_pub_keys/data.yaml")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &AggregatePubkeysTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	pk, err := bls.PublicKeyFromBytes(test.Input[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, pk2 := range test.Input[1:] {
		p, err := bls.PublicKeyFromBytes(pk2)
		if err != nil {
			t.Fatal(err)
		}
		pk.Aggregate(p)
	}

	if !bytes.Equal(test.Output, pk.Marshal()) {
		t.Fatal("Output does not equal marshaled aggregated public " +
			"key bytes")
	}
}
