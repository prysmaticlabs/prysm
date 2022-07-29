package logruswitherror

import (
	"testing"

	"github.com/prysmaticlabs/prysm/build/bazel"
	"golang.org/x/tools/go/analysis/analysistest"
)

func init() {
	if bazel.BuiltWithBazel() {
		bazel.SetGoEnv()
	}
}

func TestAnalyzer(t *testing.T) {
	// TODO: Need to review how cockroachDB achieves these results.
	testdata := bazel.TestDataPath(t)
	analysistest.TestData = func() string { return testdata }
	analysistest.Run(t, testdata, Analyzer, "a")
}
