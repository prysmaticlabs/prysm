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
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

type ResolvePkgFunc func(importPath string) string

// Copy and pasted from golang.org/x/tools/go/packages
type FlatPackagesError struct {
	Pos  string // "file:line:col" or "file:line" or "" or "-"
	Msg  string
	Kind FlatPackagesErrorKind
}

type FlatPackagesErrorKind int

const (
	UnknownError FlatPackagesErrorKind = iota
	ListError
	ParseError
	TypeError
)

func (err FlatPackagesError) Error() string {
	pos := err.Pos
	if pos == "" {
		pos = "-" // like token.Position{}.String()
	}
	return pos + ": " + err.Msg
}

// FlatPackage is the JSON form of Package
// It drops all the type and syntax fields, and transforms the Imports
type FlatPackage struct {
	ID              string
	Name            string              `json:",omitempty"`
	PkgPath         string              `json:",omitempty"`
	Errors          []FlatPackagesError `json:",omitempty"`
	GoFiles         []string            `json:",omitempty"`
	CompiledGoFiles []string            `json:",omitempty"`
	OtherFiles      []string            `json:",omitempty"`
	ExportFile      string              `json:",omitempty"`
	Imports         map[string]string   `json:",omitempty"`
	Standard        bool                `json:",omitempty"`
}

type (
	PackageFunc      func(pkg *FlatPackage)
	PathResolverFunc func(path string) string
)

func resolvePathsInPlace(prf PathResolverFunc, paths []string) {
	for i, path := range paths {
		paths[i] = prf(path)
	}
}

func WalkFlatPackagesFromJSON(jsonFile string, onPkg PackageFunc) error {
	f, err := os.Open(jsonFile)
	if err != nil {
		return fmt.Errorf("unable to open package JSON file: %w", err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	for decoder.More() {
		pkg := &FlatPackage{}
		if err := decoder.Decode(&pkg); err != nil {
			return fmt.Errorf("unable to decode package in %s: %w", f.Name(), err)
		}

		onPkg(pkg)
	}
	return nil
}

func (fp *FlatPackage) ResolvePaths(prf PathResolverFunc) error {
	resolvePathsInPlace(prf, fp.CompiledGoFiles)
	resolvePathsInPlace(prf, fp.GoFiles)
	resolvePathsInPlace(prf, fp.OtherFiles)
	fp.ExportFile = prf(fp.ExportFile)
	return nil
}

// FilterFilesForBuildTags filters the source files given the current build
// tags.
func (fp *FlatPackage) FilterFilesForBuildTags() {
	fp.GoFiles = filterSourceFilesForTags(fp.GoFiles)
	fp.CompiledGoFiles = filterSourceFilesForTags(fp.CompiledGoFiles)
}

func (fp *FlatPackage) filterTestSuffix(files []string) (err error, testFiles []string, xTestFiles, nonTestFiles []string) {
	for _, filename := range files {
		if strings.HasSuffix(filename, "_test.go") {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, filename, nil, parser.PackageClauseOnly)
			if err != nil {
				return err, nil, nil, nil
			}
			if f.Name.Name == fp.Name {
				testFiles = append(testFiles, filename)
			} else {
				xTestFiles = append(xTestFiles, filename)
			}
		} else {
			nonTestFiles = append(nonTestFiles, filename)
		}
	}
	return
}

func (fp *FlatPackage) MoveTestFiles() *FlatPackage {
	err, tgf, xtgf, gf := fp.filterTestSuffix(fp.GoFiles)

	if err != nil {
		return nil
	}
	fp.GoFiles = append(gf, tgf...)
	fp.CompiledGoFiles = append(gf, tgf...)

	if len(xtgf) == 0 {
		return nil
	}

	newImports := make(map[string]string, len(fp.Imports))
	for k, v := range fp.Imports {
		newImports[k] = v
	}

	newImports[fp.PkgPath] = fp.ID

	// Clone package, only xtgf files
	return &FlatPackage{
		ID:              fp.ID + "_xtest",
		Name:            fp.Name + "_test",
		PkgPath:         fp.PkgPath + "_test",
		Imports:         newImports,
		Errors:          fp.Errors,
		GoFiles:         append([]string{}, xtgf...),
		CompiledGoFiles: append([]string{}, xtgf...),
		OtherFiles:      fp.OtherFiles,
		ExportFile:      fp.ExportFile,
		Standard:        fp.Standard,
	}
}

func (fp *FlatPackage) IsStdlib() bool {
	return fp.Standard
}

func (fp *FlatPackage) ResolveImports(resolve ResolvePkgFunc) error {
	// Stdlib packages are already complete import wise
	if fp.IsStdlib() {
		return nil
	}

	fset := token.NewFileSet()

	for _, file := range fp.CompiledGoFiles {
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		// If the name is not provided, fetch it from the sources
		if fp.Name == "" {
			fp.Name = f.Name.Name
		}

		for _, rawImport := range f.Imports {
			imp, err := strconv.Unquote(rawImport.Path.Value)
			if err != nil {
				continue
			}
			// We don't handle CGo for now
			if imp == "C" {
				continue
			}
			if _, ok := fp.Imports[imp]; ok {
				continue
			}

			if pkgID := resolve(imp); pkgID != "" {
				fp.Imports[imp] = pkgID
			}
		}
	}

	return nil
}

func (fp *FlatPackage) IsRoot() bool {
	return strings.HasPrefix(fp.ID, "//")
}
