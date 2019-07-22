package spectest

import (
	"io/ioutil"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

const prefix = "tests/bls/"

// Load BLS yaml from spec test bls directory. The file parameter should be in
// the format of the path starting at the bls directory.
// Example: aggregate_pubkeys/aggregate_pubkeys.yaml where the full path would
// be tests/bls/aggregate_pubkeys/aggregate_pubkeys.yaml.
func loadBlsYaml(file string) ([]byte, error) {
	filepath, err := bazel.Runfile(prefix + file)
	if err != nil {
		return []byte{}, err
	}
	return ioutil.ReadFile(filepath)
}
