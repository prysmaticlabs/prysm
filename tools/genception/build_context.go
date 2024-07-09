package main

import (
	"go/build"
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

		match, _ := buildContext.MatchFile(dir, filename)
		// MatchFile filters out anything without a file extension. In the
		// case of CompiledGoFiles (in particular gco processed files from
		// the cache), we want them.
		if match || ext == "" {
			ret = append(ret, f)
		}
	}
	return ret
}
