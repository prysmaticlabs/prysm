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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

type driverResponse struct {
	// NotHandled is returned if the request can't be handled by the current
	// driver. If an external driver returns a response with NotHandled, the
	// rest of the driverResponse is ignored, and go/packages will fallback
	// to the next driver. If go/packages is extended in the future to support
	// lists of multiple drivers, go/packages will fall back to the next driver.
	NotHandled bool

	// Compiler and Arch are the arguments pass of types.SizesFor
	// to get a types.Sizes to use when type checking.
	Compiler string
	Arch     string

	// Roots is the set of package IDs that make up the root packages.
	// We have to encode this separately because when we encode a single package
	// we cannot know if it is one of the roots as that requires knowledge of the
	// graph it is part of.
	Roots []string `json:",omitempty"`

	// Packages is the full set of packages in the graph.
	// The packages are not connected into a graph.
	// The Imports if populated will be stubs that only have their ID set.
	// Imports will be connected and then type and syntax information added in a
	// later pass (see refine).
	Packages []*FlatPackage
}

var (
	// Injected via x_defs.

	rulesGoRepositoryName string
	goDefaultAspect       = rulesGoRepositoryName + "//go/tools/gopackagesdriver:aspect.bzl%go_pkg_info_aspect"
	bazelBin              = getenvDefault("GOPACKAGESDRIVER_BAZEL", "bazel")
	bazelStartupFlags     = strings.Fields(os.Getenv("GOPACKAGESDRIVER_BAZEL_FLAGS"))
	bazelQueryFlags       = strings.Fields(os.Getenv("GOPACKAGESDRIVER_BAZEL_QUERY_FLAGS"))
	bazelQueryScope       = getenvDefault("GOPACKAGESDRIVER_BAZEL_QUERY_SCOPE", "")
	bazelBuildFlags       = strings.Fields(os.Getenv("GOPACKAGESDRIVER_BAZEL_BUILD_FLAGS"))
	workspaceRoot         = os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	additionalAspects     = strings.Fields(os.Getenv("GOPACKAGESDRIVER_BAZEL_ADDTL_ASPECTS"))
	additionalKinds       = strings.Fields(os.Getenv("GOPACKAGESDRIVER_BAZEL_KINDS"))
	emptyResponse         = &driverResponse{
		NotHandled: true,
		Compiler:   "gc",
		Arch:       runtime.GOARCH,
		Roots:      []string{},
		Packages:   []*FlatPackage{},
	}
)

func run(ctx context.Context, in io.Reader, out io.Writer, args []string) error {
	queries := args

	request, err := ReadDriverRequest(in)
	if err != nil {
		return fmt.Errorf("unable to read request: %w", err)
	}

	bazel, err := NewBazel(ctx, bazelBin, workspaceRoot, bazelStartupFlags)
	if err != nil {
		return fmt.Errorf("unable to create bazel instance: %w", err)
	}

	bazelJsonBuilder, err := NewBazelJSONBuilder(bazel, request.Tests)
	if err != nil {
		return fmt.Errorf("unable to build JSON files: %w", err)
	}

	labels, err := bazelJsonBuilder.Labels(ctx, queries)
	if err != nil {
		return fmt.Errorf("unable to lookup package: %w", err)
	}

	jsonFiles, err := bazelJsonBuilder.Build(ctx, labels, request.Mode)
	if err != nil {
		return fmt.Errorf("unable to build JSON files: %w", err)
	}

	driver, err := NewJSONPackagesDriver(jsonFiles, bazelJsonBuilder.PathResolver(), bazel.version)
	if err != nil {
		return fmt.Errorf("unable to load JSON files: %w", err)
	}

	// Note: we are returning all files required to build a specific package.
	// For file queries (`file=`), this means that the CompiledGoFiles will
	// include more than the only file being specified.
	resp := driver.GetResponse(labels)
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("unable to marshal response: %v", err)
	}
	_, err = out.Write(data)
	return err
}

func main() {
	ctx, cancel := signalContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx, os.Stdin, os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		// gopls will check the packages driver exit code, and if there is an
		// error, it will fall back to go list. Obviously we don't want that,
		// so force a 0 exit code.
		os.Exit(0)
	}
}
