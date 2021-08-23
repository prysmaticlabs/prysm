package testutil

import (
	"io/ioutil"
	"path"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/pkg/errors"
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
	if len(fileBytes) == 0 {
		return nil, errors.New("empty file")
	}
	return fileBytes, nil
}

// BazelListFiles lists all of the file names in a given directory. Excludes directories. Returns
// error on empty directory.
func BazelListFiles(filepath string) ([]string, error) {
	p, err := bazel.Runfile(filepath)
	if err != nil {
		return nil, err
	}
	d, err := ioutil.ReadDir(p)
	if err != nil {
		return nil, err
	}

	ret := make([]string, 0, len(d))

	for _, f := range d {
		if f.IsDir() {
			continue
		}
		ret = append(ret, f.Name())
	}

	if len(ret) == 0 {
		return nil, errors.New("empty directory")
	}

	return ret, nil
}
