package interop

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-yaml/yaml"
)

type TestCase struct {
	Privkey string `yaml:"privkey"`
}

type KeyTest struct {
	TestCases []*TestCase `yaml:"test_cases"`
}

func TestKeyGenerator(t *testing.T) {
	path, err := bazel.Runfile("keygen_test_vector.yaml")
	if err != nil {
		t.Fatal(err)
	}
	file, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	testCases := &KeyTest{}
	if err := yaml.Unmarshal(file, testCases); err != nil {
		t.Fatal(err)
	}
	priv, _, err := DeterministicallyGenerateKeys(0, 1000)
	if err != nil {
		t.Error(err)
	}
	// cross-check with the first 1000 keys generated from the python spec
	for i, key := range priv {
		hexKey := testCases.TestCases[i].Privkey
		nKey, err := hexutil.Decode("0x" + hexKey)
		if err != nil {
			t.Error(err)
			continue
		}
		if !bytes.Equal(key.Marshal(), nKey) {
			t.Errorf("key for index %d failed, wanted %v but got %v", i, nKey, key.Marshal())
		}
	}
}
