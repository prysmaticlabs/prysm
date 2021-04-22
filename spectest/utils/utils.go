package utils

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/ghodss/yaml"
	jsoniter "github.com/json-iterator/go"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

var json = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            true,
	ValidateJsonRawMessage: true,
	TagKey:                 "spec-name",
}.Froze()

// UnmarshalYaml using a customized json encoder that supports "spec-name"
// override tag.
func UnmarshalYaml(y []byte, dest interface{}) error {
	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, dest)
}

// TestFolders sets the proper config and returns the result of ReadDir
// on the passed in eth2-spec-tests directory along with its path.
func TestFolders(t testing.TB, config, forkOrPhase, folderPath string) ([]os.FileInfo, string) {
	testsFolderPath := path.Join("tests", config, forkOrPhase, folderPath)
	filepath, err := bazel.Runfile(testsFolderPath)
	require.NoError(t, err)
	testFolders, err := ioutil.ReadDir(filepath)
	require.NoError(t, err)

	return testFolders, testsFolderPath
}

// BazelDirectoryNonEmpty returns true if directory exists and is not empty.
func BazelDirectoryNonEmpty(filePath string) (bool, error) {
	p, err := bazel.Runfile(filePath)
	if err != nil {
		return false, err
	}
	fs, err := ioutil.ReadDir(p)
	if err != nil {
		return false, err
	}
	return len(fs) > 0, nil
}

// BazelFileBytes returns the byte array of the bazel file path given.
func BazelFileBytes(filePaths ...string) ([]byte, error) {
	filepath, err := bazel.Runfile(path.Join(filePaths...))
	if err != nil {
		return nil, err
	}
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return fileBytes, nil
}
