// Package spectest includes tests to ensure conformity with the eth2
// bls cryptography specification.
package spectest

import (
	"bytes"
	"encoding/hex"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSignMessageYaml(t *testing.T) {
	flags := &featureconfig.Flags{}
	reset := featureconfig.InitWithReset(flags)
	t.Run("herumi", testSignMessageYaml)
	reset()

	flags.EnableBlst = true
	reset = featureconfig.InitWithReset(flags)
	t.Run("blst", testSignMessageYaml)
	reset()
}

func testSignMessageYaml(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/sign/small")

	for i, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			test := &SignMsgTest{}
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
			sig := sk.Sign(msgBytes)

			if !sig.Verify(sk.PublicKey(), msgBytes) {
				t.Fatal("could not verify signature")
			}

			outputBytes, err := hex.DecodeString(test.Output[2:])
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}

			if !bytes.Equal(outputBytes, sig.Marshal()) {
				t.Fatalf("Test Case %d: Signature does not match the expected output. "+
					"Expected %#x but received %#x", i, outputBytes, sig.Marshal())
			}
			t.Log("Success")
		})
	}
}
