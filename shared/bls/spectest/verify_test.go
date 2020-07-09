package spectest

import (
	"encoding/hex"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestVerifyMessageYaml(t *testing.T) {
	flags := &featureconfig.Flags{}
	reset := featureconfig.InitWithReset(flags)
	t.Run("herumi", testVerifyMessageYaml)
	reset()

	flags.EnableBlst = true
	reset = featureconfig.InitWithReset(flags)
	t.Run("blst", testVerifyMessageYaml)
	reset()
}

func testVerifyMessageYaml(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/verify/small")

	for i, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			test := &VerifyMsgTest{}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			pkBytes, err := hex.DecodeString(test.Input.Pubkey[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			pk, err := bls.PublicKeyFromBytes(pkBytes)
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
				if test.Output == false {
					return
				}
				t.Fatalf("Cannot unmarshal input to signature: %v", err)
			}

			verified := sig.Verify(pk, msgBytes)
			if verified != test.Output {
				t.Fatalf("Signature does not match the expected verification output. "+
					"Expected %#v but received %#v for test case %d", test.Output, verified, i)
			}
			t.Log("Success")
		})
	}
}
