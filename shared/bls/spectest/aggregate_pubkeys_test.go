package spectest

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestAggregatePubkeysYaml(t *testing.T) {
	file, err := testutil.BazelFileBytes("tests/general/phase0/bls/aggregate_pubkeys/small/agg_pub_keys/data.yaml")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &AggregatePubkeysTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	pubBytes, err := hex.DecodeString(test.Input[0][2:])
	if err != nil {
		t.Fatalf("Cannot decode string to bytes: %v", err)
	}
	pk, err := bls.PublicKeyFromBytes(pubBytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, pk2 := range test.Input[1:] {
		pubBytes2, err := hex.DecodeString(pk2[2:])
		if err != nil {
			t.Fatalf("Cannot decode string to bytes: %v", err)
		}
		p, err := bls.PublicKeyFromBytes(pubBytes2)
		if err != nil {
			t.Fatal(err)
		}
		pk.Aggregate(p)
	}

	outputBytes, err := hex.DecodeString(test.Output[2:])
	if err != nil {
		t.Fatalf("Cannot decode string to bytes: %v", err)
	}
	if !bytes.Equal(outputBytes, pk.Marshal()) {
		t.Fatal("Output does not equal marshaled aggregated public " +
			"key bytes")
	}
}
