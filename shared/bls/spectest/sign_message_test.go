package spectest

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

func TestSignMessageYaml(t *testing.T) {
	file, err := loadBlsYaml("sign_msg/sign_msg.yaml")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &SignMessageTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	for i, tt := range test.TestCases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			sk, err := bls.SecretKeyFromBytes(tt.Input.Privkey)
			if err != nil {
				t.Fatalf("Cannot unmarshal input to secret key: %v", err)
			}

			sig := sk.Sign(tt.Input.Message, tt.Input.Domain)
			if !bytes.Equal(tt.Output, sig.Marshal()) {
				t.Logf("Domain=%d", tt.Input.Domain)
				t.Fatalf("Signature does not match the expected output. "+
					"Expected %#x but received %#x", tt.Output, sig.Marshal())
			}
			t.Logf("Success. Domain=%d", tt.Input.Domain)
		})
	}
}
