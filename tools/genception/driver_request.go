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
	"encoding/json"
	"fmt"
	"io"
)

// From https://pkg.go.dev/golang.org/x/tools/go/packages#LoadMode
type LoadMode int

// Only NeedExportsFile is needed in our case
const (
	// NeedName adds Name and PkgPath.
	NeedName LoadMode = 1 << iota

	// NeedFiles adds GoFiles and OtherFiles.
	NeedFiles

	// NeedCompiledGoFiles adds CompiledGoFiles.
	NeedCompiledGoFiles

	// NeedImports adds Imports. If NeedDeps is not set, the Imports field will contain
	// "placeholder" Packages with only the ID set.
	NeedImports

	// NeedDeps adds the fields requested by the LoadMode in the packages in Imports.
	NeedDeps

	// NeedExportsFile adds ExportFile.
	NeedExportFile

	// NeedTypes adds Types, Fset, and IllTyped.
	NeedTypes

	// NeedSyntax adds Syntax.
	NeedSyntax

	// NeedTypesInfo adds TypesInfo.
	NeedTypesInfo

	// NeedTypesSizes adds TypesSizes.
	NeedTypesSizes

	// typecheckCgo enables full support for type checking cgo. Requires Go 1.15+.
	// Modifies CompiledGoFiles and Types, and has no effect on its own.
	typecheckCgo

	// NeedModule adds Module.
	NeedModule
)

// Deprecated: NeedExportsFile is a historical misspelling of NeedExportFile.
const NeedExportsFile = NeedExportFile

// From https://github.com/golang/tools/blob/v0.1.0/go/packages/external.go#L32
// Most fields are disabled since there is no need for them
type DriverRequest struct {
	Mode LoadMode `json:"mode"`
	// Env specifies the environment the underlying build system should be run in.
	// Env []string `json:"env"`
	// BuildFlags are flags that should be passed to the underlying build system.
	// BuildFlags []string `json:"build_flags"`
	// Tests specifies whether the patterns should also return test packages.
	Tests bool `json:"tests"`
	// Overlay maps file paths (relative to the driver's working directory) to the byte contents
	// of overlay files.
	// Overlay map[string][]byte `json:"overlay"`
}

func ReadDriverRequest(r io.Reader) (*DriverRequest, error) {
	req := &DriverRequest{}
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return nil, fmt.Errorf("unable to decode driver request: %w", err)
	}
	return req, nil
}
