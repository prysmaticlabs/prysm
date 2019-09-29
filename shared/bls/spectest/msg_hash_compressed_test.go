package spectest

import (
	"bytes"
	"path"
	"testing"

	bls12 "github.com/kilic/bls12-381"
	blssig "github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/yaml.v2"
)

type msgHashCompressedTest struct {
	Input struct {
		Message string `yaml:"message"`
		Domain  string `yaml:"domain"`
	} `yaml:"input"`
	Output [2]string `yaml:"output"`
}

func TestMsgHashCompressed(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/msg_hash_compressed/small")
	g2 := bls12.NewG2(nil)
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			test := &msgHashCompressedTest{}
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatal(err)
			}
			expectedOutputString := toBytes(48, test.Output[0], test.Output[1])
			msg := toBytes(32, test.Input.Message)
			domain := toBytes(8, test.Input.Domain)
			msgHashBytes := blssig.HashWithDomain(bytesutil.ToBytes32(msg), bytesutil.ToBytes8(domain))
			msgHashG2Point := g2.MapToPoint(msgHashBytes)
			if !bytes.Equal(expectedOutputString[:], g2.ToCompressed(msgHashG2Point)) {
				t.Fatal("msg hash compressed fails\n", folder.Name())
			}
		})
	}
}
