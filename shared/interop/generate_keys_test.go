package interop_test

import (
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

type TestCase struct {
	Privkey string `yaml:"privkey"`
}

type KeyTest struct {
	TestCases []*TestCase `yaml:"test_cases"`
}

func TestKeyGenerator(t *testing.T) {
	path, err := bazel.Runfile("keygen_test_vector.yaml")
	require.NoError(t, err)
	file, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	testCases := &KeyTest{}
	require.NoError(t, yaml.Unmarshal(file, testCases))
	priv, _, err := interop.DeterministicallyGenerateKeys(0, 1000)
	require.NoError(t, err)
	// cross-check with the first 1000 keys generated from the python spec
	for i, key := range priv {
		hexKey := testCases.TestCases[i].Privkey
		nKey, err := hexutil.Decode("0x" + hexKey)
		if err != nil {
			t.Error(err)
			continue
		}
		assert.DeepEqual(t, key.Marshal(), nKey)
	}
}
