package fuzz_test

import (
	"io/ioutil"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks/fuzz"
)

// Test any crashed input as needed.
func TestFuzzRegression(t *testing.T) {
	runfiles, err := bazel.ListRunfiles()
	if err != nil {
		t.Fatal(err)
	}
	var filepaths []string
	for _, file := range runfiles {
		if file.ShortPath[len(file.ShortPath)-4:] == ".ssz" {
			filepaths = append(filepaths, file.Path)
		}
	}

	if len(filepaths) == 0 {
		t.Fatal("no files to run")
	}
	for _, file := range filepaths {
		t.Run(file, func(t *testing.T) {
			b, err := ioutil.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := fuzz.BeaconFuzz(b); !ok {
				t.Fatal("not ok")
			}
		})
	}
}