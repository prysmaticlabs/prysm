package spectest

import (
	"bytes"
	"encoding/binary"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/yaml.v2"
)

type testVectorSignMessage struct {
	Input struct {
		Secret  string `yaml:"privkey"`
		Message string `yaml:"message"`
		Domain  string `yaml:"domain"`
	} `yaml:"input"`
	Output string `yaml:"output"`
}

func TestSignMessage(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/sign_msg/small")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			test := &testVectorSignMessage{}
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatal(err)
			}
			expectedOutputString := toBytes(96, test.Output)
			msg := toBytes(32, test.Input.Message)
			domain := binary.LittleEndian.Uint64(toBytes(8, test.Input.Domain))
			secretKey, err := bls.SecretKeyFromBytes(toBytes(32, test.Input.Secret))
			if err != nil {
				t.Fatal(err)
			}
			signature := secretKey.Sign(msg, domain)
			if !bytes.Equal(expectedOutputString[:], signature.Marshal()) {
				t.Fatal("msg sign fails\n", folder.Name())
			}
		})
	}
}
