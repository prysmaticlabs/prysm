package fuzz

import (
	"io/ioutil"
	"strings"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

const PanicOnError = "false"

func init() {
	featureconfig.Init(&featureconfig.Flags{
		SkipBLSVerify: true,
	})
}

func fail(err error) ([]byte, bool) {
	if strings.ToLower(PanicOnError) == "true" {
		panic(err)
	}
	return nil, false
}

func bazelFileBytes(path string) ([]byte, error) {
	filepath, err := bazel.Runfile(path)
	if err != nil {
		return nil, err
	}
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return fileBytes, nil
}
