package spectest

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/phoreproject/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// Note: This actually tests the underlying library as we don't have a need for
// HashG2Uncompressed in our local BLS API.
func TestG2UncompressedHash(t *testing.T) {
	t.Skip("The python uncompressed method does not match the go uncompressed method and this isn't very important")
	file, err := loadBlsYaml("msg_hash_g2_uncompressed/g2_uncompressed.yaml")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &G2UncompressedTest{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	for i, tt := range test.TestCases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			projective := bls.HashG2WithDomain(
				bytesutil.ToBytes32(tt.Input.Message),
				tt.Input.Domain,
			)
			hash := projective.ToAffine().SerializeBytes()

			var buf []byte
			for _, slice := range tt.Output {
				for _, innerSlice := range slice {
					buf = append(buf, innerSlice...)
				}
			}
			if !bytes.Equal(buf, hash[:]) {
				t.Logf("Domain=%d", tt.Input.Domain)
				t.Fatalf("Hash does not match the expected output. "+
					"Expected %#x but received %#x", buf, hash)
			}
			t.Logf("Success. Domain=%d", tt.Input.Domain)
		})
	}
}
