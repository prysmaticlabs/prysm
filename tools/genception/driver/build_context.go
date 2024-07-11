package driver

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

var buildContext = makeBuildContext()

func makeBuildContext() *build.Context {
	bctx := build.Default
	bctx.BuildTags = strings.Split(getenvDefault("GOTAGS", ""), ",")

	return &bctx
}

func filterSourceFilesForTags(files []string) []string {
	ret := make([]string, 0, len(files))

	for _, f := range files {
		dir, filename := filepath.Split(f)
		ext := filepath.Ext(f)

		match, err := buildContext.MatchFile(dir, filename)
		if err != nil {
			log.WithError(err).WithField("file", f).Warn("error matching file")
		}
		// MatchFile filters out anything without a file extension. In the
		// case of CompiledGoFiles (in particular gco processed files from
		// the cache), we want them.
		if match || ext == "" {
			ret = append(ret, f)
		}
	}
	return ret
}
func getenvDefault(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultValue
}
