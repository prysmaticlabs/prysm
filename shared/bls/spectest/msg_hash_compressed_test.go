package spectest

import (
	"bytes"
	"encoding/hex"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/phoreproject/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

// Note: This actually tests the underlying library as we don't have a need for
// HashG2Compressed in our local BLS API.
func TestMsgHashCompressed(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/msg_hash_compressed/small")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			test := &MsgHashCompressedTest{}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			msgBytes, err := hex.DecodeString(test.Input.Message[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			domain, err := hex.DecodeString(test.Input.Domain[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			projective := bls.HashG2WithDomain(
				bytesutil.ToBytes32(msgBytes),
				bytesutil.ToBytes8(domain),
			)
			hash := bls.CompressG2(projective.ToAffine())

			var buf []byte
			for _, innerString := range test.Output {
				slice, err := hex.DecodeString(innerString[2:])
				if err != nil {
					t.Fatalf("Cannot decode string to bytes: %v", err)
				}
				buf = append(buf, slice...)
			}
			if !bytes.Equal(buf, hash[:]) {
				t.Logf("Domain=%d", domain)
				t.Fatalf("Hash does not match the expected output. "+
					"Expected %#x but received %#x", buf, hash)
			}
			t.Logf("Success. Domain=%d", domain)
		})
	}
}
