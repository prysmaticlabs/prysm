package spectest

import (
	"bytes"
	"encoding/binary"
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
			file, err := loadBlsYaml(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			test := &MsgHashCompressedTest{}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, test.Input.Domain)

			projective := bls.HashG2WithDomain(
				bytesutil.ToBytes32(test.Input.Message),
				bytesutil.ToBytes8(b),
			)
			hash := bls.CompressG2(projective.ToAffine())

			var buf []byte
			for _, slice := range test.Output {
				buf = append(buf, slice...)
			}
			if !bytes.Equal(buf, hash[:]) {
				t.Logf("Domain=%d", test.Input.Domain)
				t.Fatalf("Hash does not match the expected output. "+
					"Expected %#x but received %#x", buf, hash)
			}
			t.Logf("Success. Domain=%d", test.Input.Domain)
		})
	}
}
