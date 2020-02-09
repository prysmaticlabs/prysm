package spectest

import (
	"encoding/hex"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	bls "github.com/prysmaticlabs/prysm/shared/bls_new"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestVerifyMessageYaml(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/sign/small")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			test := &VerifyMsgTest{}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			pkBytes, err := hex.DecodeString(test.Input.Privkey[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			sk, err := bls.SecretKeyFromBytes(pkBytes)
			if err != nil {
				t.Fatalf("Cannot unmarshal input to secret key: %v", err)
			}

			msgBytes, err := hex.DecodeString(test.Input.Message[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}

			sigBytes, err := hex.DecodeString(test.Input.Signature[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			sig, err := bls.SignatureFromBytes(sigBytes)
			if err != nil {
				t.Fatalf("Cannot unmarshal input to signature: %v", err)
			}

			outputBytes, err := hex.DecodeString(test.Output[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}

			if sig.Verify(sk.Publickey(), msgBytes) != true {
				t.Fatalf("Signature does not match the expected verification output. "+
					"Expected %#x but received %#x", outputBytes, sig.Marshal())
			}
			t.Log("Success")
		})
	}
}
