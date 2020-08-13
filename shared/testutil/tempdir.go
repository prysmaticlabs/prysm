package testutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/prysmaticlabs/prysm/shared/rand"
)

// TempDir returns a directory path for temporary test storage.
func TempDir() string {
	d := os.Getenv("TEST_TMPDIR")

	// If the test is not run via bazel, the environment var won't be set.
	if d == "" {
		return os.TempDir()
	}
	return d
}

// RandDir returns a random temporary directory.
func RandDir() (string, error) {
	randPath := rand.NewDeterministicGenerator().Int()
	randDir := filepath.Join(TempDir(), fmt.Sprintf("/%d", randPath))
	if err := os.MkdirAll(randDir, os.ModePerm); err != nil {
		return "", err
	}
	return randDir, nil
}
