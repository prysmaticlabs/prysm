package spectest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestG2CompressedHash(t *testing.T) {
	file, err := ioutil.ReadFile("g2_compressed.yaml")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	test := &G2Compressed{}
	if err := yaml.Unmarshal(file, test); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	for i, tt := range test.TestCases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			hash := bls.HashG2WithDomainCompressed(bytesutil.ToBytes32(tt.Input.Message), tt.Input.Domain)
			var buf []byte
			for _, slice := range tt.Output {
				buf = append(buf, slice...)
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
