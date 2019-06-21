package spectest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestPrivToPubYaml(t *testing.T) {
	file, err := ioutil.ReadFile("priv_to_pub_formatted.yaml")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &PrivToPubTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	for i, tt := range test.TestCases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			sk, err := bls.SecretKeyFromBytes(tt.Input)
			if err != nil {
				t.Fatalf("Cannot unmarshal input to secret key: %v", err)
			}
			if !bytes.Equal(tt.Output, sk.PublicKey().Marshal()) {
				t.Fatal("Output does not marshalled public key bytes")
			}
		})
	}
}
