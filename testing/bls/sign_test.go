package bls

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/v4/testing/bls/utils"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestSign(t *testing.T) {
	t.Run("blst", testSign)
}

func testSign(t *testing.T) {
	fNames, fContent := utils.RetrieveFiles("sign", t)

	for i, file := range fNames {
		t.Run(file, func(t *testing.T) {
			test := &SignMsgTest{}
			require.NoError(t, yaml.Unmarshal(fContent[i], test))
			pkBytes, err := hex.DecodeString(test.Input.Privkey[2:])
			require.NoError(t, err)
			sk, err := bls.SecretKeyFromBytes(pkBytes)
			if err != nil {
				if test.Output == "" &&
					(errors.Is(err, common.ErrZeroKey) || errors.Is(err, common.ErrSecretUnmarshal)) {
					return
				}
				t.Fatalf("cannot unmarshal secret key: %v", err)
			}
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
