package spectest

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSignMessageYaml(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/sign_msg/small")

	for _, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := loadBlsYaml(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			test := &SignMsgTest{}
			if err := yaml.Unmarshal(file, test); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			pkBytes, err := hexutil.Decode(test.Input.Privkey)
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			sk, err := bls.SecretKeyFromBytes(pkBytes)
			if err != nil {
				t.Fatalf("Cannot unmarshal input to secret key: %v", err)
			}

			msgBytes, err := hexutil.Decode(test.Input.Message)
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			domain, err := hexutil.DecodeUint64(test.Input.Domain)
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			sig := sk.Sign(msgBytes, domain)

			outputBytes, err := hexutil.Decode(test.Output)
			if err != nil {
				t.Fatalf("Cannot decode string to bytes: %v", err)
			}
			if !bytes.Equal(outputBytes, sig.Marshal()) {
				t.Logf("Domain=%d", domain)
				t.Fatalf("Signature does not match the expected output. "+
					"Expected %#x but received %#x", outputBytes, sig.Marshal())
			}
			t.Logf("Success. Domain=%d", domain)
		})
	}
}
