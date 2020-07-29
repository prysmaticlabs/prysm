package spectest

import (
	"encoding/hex"
	"path"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bls/iface"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestAggregateVerifyYaml(t *testing.T) {
	flags := &featureconfig.Flags{}
	reset := featureconfig.InitWithReset(flags)
	t.Run("herumi", testAggregateVerifyYaml)
	reset()

	flags.EnableBlst = true
	reset = featureconfig.InitWithReset(flags)
	t.Run("blst", testAggregateVerifyYaml)
	reset()
}

func testAggregateVerifyYaml(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/aggregate_verify/small")

	for i, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			test := &AggregateVerifyTest{}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			pubkeys := make([]iface.PublicKey, 0, len(test.Input.Pubkeys))
			msgs := make([][32]byte, 0, len(test.Input.Messages))
			for _, pubKey := range test.Input.Pubkeys {
				pkBytes, err := hex.DecodeString(pubKey[2:])
				if err != nil {
					t.Fatalf("Cannot decode string to bytes: %v", err)
				}
				pk, err := bls.PublicKeyFromBytes(pkBytes)
				if err != nil {
					t.Fatalf("Cannot unmarshal input to secret key: %v", err)
				}
				pubkeys = append(pubkeys, pk)
			}
			for _, msg := range test.Input.Messages {
				msgBytes, err := hex.DecodeString(msg[2:])
				if err != nil {
					t.Fatalf("Cannot decode string to bytes: %v", err)
				}
				if len(msgBytes) != 32 {
					t.Fatalf("Message: %#x is not 32 bytes", msgBytes)
				}
				msgs = append(msgs, bytesutil.ToBytes32(msgBytes))
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

			verified := sig.AggregateVerify(pubkeys, msgs)
			if verified != test.Output {
				t.Fatalf("Signature does not match the expected verification output. "+
					"Expected %#v but received %#v for test case %d", test.Output, verified, i)
			}
		})
	}
}
