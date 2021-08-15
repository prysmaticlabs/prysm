package testutil

import (
	"io/ioutil"
	"path"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

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
	fileBytes, err := ioutil.ReadFile(filepath) // #nosec G304
	if err != nil {
		return nil, err
	}
	return fileBytes, nil
}
