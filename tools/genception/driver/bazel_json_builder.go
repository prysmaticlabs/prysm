// Copyright 2021 The Bazel Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"os"
	"strings"
)

var RulesGoStdlibLabel = "@io_bazel_rules_go//:stdlib"

/*
type BazelJSONBuilder struct {
	packagesBaseDir string
	includeTests    bool
}


var _defaultKinds = []string{"go_library", "go_test", "go_binary"}

var externalRe = regexp.MustCompile(`.*\/external\/([^\/]+)(\/(.*))?\/([^\/]+.go)`)

func (b *BazelJSONBuilder) fileQuery(filename string) string {
	label := filename

	if strings.HasPrefix(filename, "./") {
		label = strings.TrimPrefix(filename, "./")
	}

	if matches := externalRe.FindStringSubmatch(filename); len(matches) == 5 {
		// if filepath is for a third party lib, we need to know, what external
		// library this file is part of.
		matches = append(matches[:2], matches[3:]...)
		label = fmt.Sprintf("@%s//%s", matches[1], strings.Join(matches[2:], ":"))
	}

	relToBin, err := filepath.Rel(b.bazel.info["output_path"], filename)
	if err == nil && !strings.HasPrefix(relToBin, "../") {
		parts := strings.SplitN(relToBin, string(filepath.Separator), 3)
		relToBin = parts[2]
		// We've effectively converted filename from bazel-bin/some/path.go to some/path.go;
		// Check if a BUILD.bazel files exists under this dir, if not walk up and repeat.
		relToBin = filepath.Dir(relToBin)
		_, err = os.Stat(filepath.Join(b.bazel.WorkspaceRoot(), relToBin, "BUILD.bazel"))
		for errors.Is(err, os.ErrNotExist) && relToBin != "." {
			relToBin = filepath.Dir(relToBin)
			_, err = os.Stat(filepath.Join(b.bazel.WorkspaceRoot(), relToBin, "BUILD.bazel"))
		}

		if err == nil {
			// return package path found and build all targets (codegen doesn't fall under go_library)
			// Otherwise fallback to default
			if relToBin == "." {
				relToBin = ""
			}
			label = fmt.Sprintf("//%s:all", relToBin)
		}
	}

	return label
}

func isLocalImport(path string) bool {
	return path == "." || path == ".." ||
		strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") ||
		filepath.IsAbs(path)
}

func NewBazelJSONBuilder(includeTests bool) (*BazelJSONBuilder, error) {
	return &BazelJSONBuilder{
		includeTests:    includeTests,
	}, nil
}

func (b *BazelJSONBuilder) Labels(ctx context.Context, requests []string) ([]string, error) {
	ret := make([]string, 0, len(requests))
	for _, request := range requests {
		result := ""
		if strings.HasSuffix(request, ".go") {
			f := strings.TrimPrefix(request, "file=")
			result = b.fileQuery(f)
		} else if request == "builtin" || request == "std" {
			result = fmt.Sprintf(RulesGoStdlibLabel)
		}

		if result != "" {
			ret = append(ret, result)
		}
	}
	if len(ret) == 0 {
		return []string{RulesGoStdlibLabel}, nil
	}
	return ret, nil
}

func (b *BazelJSONBuilder) PathResolver() PathResolverFunc {
	return func(p string) string {
		p = strings.Replace(p, "__BAZEL_EXECROOT__", os.Getenv("PWD"), 1)
		p = strings.Replace(p, "__BAZEL_OUTPUT_BASE__", b.packagesBaseDir, 1)
		return p
	}
}
*/

func NewPathResolver() (*PathResolver, error) {
	outBase, err := PackagesBaseFromEnv()
	if err != nil {
		return nil, err
	}
	return &PathResolver{
		execRoot:   os.Getenv("PWD"),
		outputBase: outBase,
	}, nil
}

type PathResolver struct {
	outputBase string
	execRoot   string
}

const (
	prefixExecRoot   = "__BAZEL_EXECROOT__"
	prefixOutputBase = "__BAZEL_OUTPUT_BASE__"
	prefixWorkspace  = "__BAZEL_WORKSPACE__"
)

var prefixes = []string{prefixExecRoot, prefixOutputBase, prefixWorkspace}

func (r PathResolver) Resolve(path string) string {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			for _, rpl := range []string{r.execRoot, r.outputBase} {
				rp := strings.Replace(path, prefix, rpl, 1)
				_, err := os.Stat(rp)
				if err == nil {
					return rp
				}
			}
			return path
		}
	}
	log.WithField("path", path).Warn("unrecognized path prefix when resolving source paths in json import metadata")
	return path
}
