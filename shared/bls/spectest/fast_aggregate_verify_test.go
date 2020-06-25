package spectest

import (
	"encoding/hex"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bls/iface"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestFastAggregateVerifyYaml(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/fast_aggregate_verify/small")

	for i, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			test := &FastAggregateVerifyTest{}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			pubkeys := make([]iface.PublicKey, len(test.Input.Pubkeys))
			for j, raw := range test.Input.Pubkeys {
				pkBytes, err := hex.DecodeString(raw[2:])
				if err != nil {
					t.Fatalf("Cannot decode string to bytes: %v", err)
				}
				pk, err := bls.PublicKeyFromBytes(pkBytes)
				if err != nil {
					t.Fatalf("Cannot unmarshal input to secret key: %v", err)
				}
				pubkeys[j] = pk
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

			verified := sig.FastAggregateVerify(pubkeys, bytesutil.ToBytes32(msgBytes))
			if verified != test.Output {
				t.Fatalf("Signature does not match the expected verification output. "+
					"Expected %#v but received %#v for test case %d", test.Output, verified, i)
			}
			t.Log("Success")
		})
	}
}
