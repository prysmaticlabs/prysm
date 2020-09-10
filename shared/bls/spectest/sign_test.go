// Package spectest includes tests to ensure conformity with the eth2
// bls cryptography specification.
package spectest

import (
	"bytes"
	"encoding/hex"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSignMessageYaml(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/sign/small")

	for i, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			require.NoError(t, err)
			test := &SignMsgTest{}
			require.NoError(t, yaml.Unmarshal(file, test))
			pkBytes, err := hex.DecodeString(test.Input.Privkey[2:])
			require.NoError(t, err)
			sk, err := bls.SecretKeyFromBytes(pkBytes)
			require.NoError(t, err)

			msgBytes, err := hex.DecodeString(test.Input.Message[2:])
			require.NoError(t, err)
			sig := sk.Sign(msgBytes)

			if !sig.Verify(sk.PublicKey(), msgBytes) {
				t.Fatal("could not verify signature")
			}

			outputBytes, err := hex.DecodeString(test.Output[2:])
			require.NoError(t, err)

			if !bytes.Equal(outputBytes, sig.Marshal()) {
				t.Fatalf("Test Case %d: Signature does not match the expected output. "+
					"Expected %#x but received %#x", i, outputBytes, sig.Marshal())
			}
			t.Log("Success")
		})
	}
}
