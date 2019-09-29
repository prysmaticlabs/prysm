package spectest

import (
	"bytes"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/yaml.v2"
)

type privToPubTest struct {
	Input  string `yaml:"input"`
	Output string `yaml:"output"`
}

func TestPrivToPub(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/priv_to_pub/small")
	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			test := &privToPubTest{}
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatal(err)
			}
			expectedOutputString := toBytes(48, test.Output)
			secretKey, err := bls.SecretKeyFromBytes(toBytes(32, test.Input))
			if err != nil {
				t.Fatal(err)
			}
			publicKey := secretKey.PublicKey()
			if !bytes.Equal(expectedOutputString, publicKey.Marshal()) {
				t.Fatal("priv to pub fails\n", folder.Name())
			}
		})
	}
}
