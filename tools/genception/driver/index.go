package driver

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

const (
	ENV_JSON_INDEX_PATH = "PACKAGE_JSON_INVENTORY"
	ENV_PACKAGES_BASE   = "PACKAGES_BASE"
)

var ErrUnsetEnvVar = errors.New("required env var not set")

// LoadJsonListing reads the list of json package index files created by the bazel gopackagesdriver aspect:
// https://github.com/bazelbuild/rules_go/blob/master/go/tools/gopackagesdriver/aspect.bzl
// This list is serialized as a []string paths, relative to the bazel exec root.
func LoadJsonListing() ([]string, error) {
	path, err := JsonIndexPathFromEnv()
	if err != nil {
		return nil, err
	}
	return ReadJsonIndex(path)
}

func ReadJsonIndex(path string) ([]string, error) {
	um := make([]string, 0)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &um); err != nil {
		return nil, err
	}

	return um, nil
}

// JsonIndexPathFromEnv reads the path to the json index file from the environment.
func JsonIndexPathFromEnv() (string, error) {
	p := os.Getenv(ENV_JSON_INDEX_PATH)
	if p == "" {
		return "", errors.Wrap(ErrUnsetEnvVar, ENV_JSON_INDEX_PATH)
	}
	return p, nil
}

func PackagesBaseFromEnv() (string, error) {
	p := os.Getenv(ENV_PACKAGES_BASE)
	if p == "" {
		return "", errors.Wrap(ErrUnsetEnvVar, ENV_PACKAGES_BASE)
	}
	return p, nil
}
