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
	"fmt"
	"runtime"
)

type JSONPackagesDriver struct {
	registry *PackageRegistry
}

func NewJSONPackagesDriver(jsonFiles []string, prf PathResolverFunc) (*JSONPackagesDriver, error) {
	jpd := &JSONPackagesDriver{
		registry: NewPackageRegistry(),
	}

	for _, f := range jsonFiles {
		if err := WalkFlatPackagesFromJSON(f, func(pkg *FlatPackage) {
			jpd.registry.Add(pkg)
		}); err != nil {
			return nil, fmt.Errorf("unable to walk json: %w", err)
		}
	}

	if err := jpd.registry.ResolvePaths(prf); err != nil {
		return nil, fmt.Errorf("unable to resolve paths: %w", err)
	}

	if err := jpd.registry.ResolveImports(); err != nil {
		return nil, fmt.Errorf("unable to resolve paths: %w", err)
	}

	return jpd, nil
}

func (b *JSONPackagesDriver) Handle(req *DriverRequest, queries []string) *driverResponse {
	r, p := b.registry.Query(req, queries)
	return &driverResponse{
		NotHandled: false,
		Compiler:   "gc",
		Arch:       runtime.GOARCH,
		Roots:      r,
		Packages:   p,
	}
}

func (b *JSONPackagesDriver) GetResponse(labels []string) *driverResponse {
	rootPkgs, packages := b.registry.Match(labels)

	return &driverResponse{
		NotHandled: false,
		Compiler:   "gc",
		Arch:       runtime.GOARCH,
		Roots:      rootPkgs,
		Packages:   packages,
	}
}

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
