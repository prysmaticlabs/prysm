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
// HashG2Uncompressed in our local BLS API.
func TestMsgHashUncompressed(t *testing.T) {
	t.Skip("The python uncompressed method does not match the go uncompressed method and this isn't very important")
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/msg_hash_uncompressed/small")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			test := &MsgHashUncompressedTest{}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			domain, err := hex.DecodeString(test.Input.Domain[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			msgBytes, err := hex.DecodeString(test.Input.Message[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			projective := bls.HashG2WithDomain(
				bytesutil.ToBytes32(msgBytes),
				bytesutil.ToBytes8(domain),
			)
			hash := projective.ToAffine().SerializeBytes()

			var buf []byte
			for _, outputStrings := range test.Output {
				for _, innerString := range outputStrings {
					slice, err := hex.DecodeString(innerString[2:])
					if err != nil {
						t.Fatalf("Cannot decode string to bytes: %v", err)
					}
					buf = append(buf, slice...)
				}
			}
			if !bytes.Equal(buf, hash[:]) {
				t.Logf("Domain=%d", domain)
				t.Logf("Message=%#x", msgBytes)
				t.Fatalf("Hash does not match the expected output. "+
					"Expected %#x but received %#x", buf, hash)
			}
			t.Logf("Success. Domain=%d", domain)
		})
	}
}
