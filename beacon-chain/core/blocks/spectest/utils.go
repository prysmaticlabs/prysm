package spectest

import (
	"io/ioutil"
	"path"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

// SSZFileBytes returns the unmarshalled SSZ interface at the passed in path.
func SSZFileBytes(folderPath string, testName string, filename string) ([]byte, error) {
	filepath, err := bazel.Runfile(path.Join(folderPath, testName, filename))
	if err != nil {
		return nil, err
	}
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return fileBytes, nil
}
