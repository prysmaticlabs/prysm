package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel_testing"
)

type response struct {
	Roots    []string `json:",omitempty"`
	Packages []*FlatPackage
}

func TestMain(m *testing.M) {
	bazel_testing.TestMain(m, bazel_testing.Args{
		Main: `
-- BUILD.bazel --
load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "hello",
    srcs = ["hello.go"],
    importpath = "example.com/hello",
    visibility = ["//visibility:public"],
)

go_test(
	name = "hello_test",
	srcs = [
		"hello_test.go",
		"hello_external_test.go",
	],
	embed = [":hello"],
)

-- hello.go --
package hello

import "os"

func main() {
	fmt.Fprintln(os.Stderr, "Hello World!")
}

-- hello_test.go --
package hello

import "testing"

func TestHelloInternal(t *testing.T) {}

-- hello_external_test.go --
package hello_test

import "testing"

func TestHelloExternal(t *testing.T) {}
		`,
	})
}

const (
	osPkgID       = "@io_bazel_rules_go//stdlib:os"
	bzlmodOsPkgID = "@@io_bazel_rules_go//stdlib:os"
)

func TestBaseFileLookup(t *testing.T) {
	resp := runForTest(t, "file=hello.go")

	t.Run("roots", func(t *testing.T) {
		if len(resp.Roots) != 1 {
			t.Errorf("Expected 1 package root: %+v", resp.Roots)
			return
		}

		if !strings.HasSuffix(resp.Roots[0], "//:hello") {
			t.Errorf("Unexpected package id: %q", resp.Roots[0])
			return
		}
	})

	t.Run("package", func(t *testing.T) {
		var pkg *FlatPackage
		for _, p := range resp.Packages {
			if p.ID == resp.Roots[0] {
				pkg = p
			}
		}

		if pkg == nil {
			t.Errorf("Expected to find %q in resp.Packages", resp.Roots[0])
			return
		}

		if len(pkg.CompiledGoFiles) != 1 || len(pkg.GoFiles) != 1 ||
			path.Base(pkg.GoFiles[0]) != "hello.go" || path.Base(pkg.CompiledGoFiles[0]) != "hello.go" {
			t.Errorf("Expected to find 1 file (hello.go) in (Compiled)GoFiles:\n%+v", pkg)
			return
		}

		if pkg.Standard {
			t.Errorf("Expected package to not be Standard:\n%+v", pkg)
			return
		}

		if len(pkg.Imports) != 1 {
			t.Errorf("Expected one import:\n%+v", pkg)
			return
		}

		if pkg.Imports["os"] != osPkgID && pkg.Imports["os"] != bzlmodOsPkgID {
			t.Errorf("Expected os import to map to %q or %q:\n%+v", osPkgID, bzlmodOsPkgID, pkg)
			return
		}
	})

	t.Run("dependency", func(t *testing.T) {
		var osPkg *FlatPackage
		for _, p := range resp.Packages {
			if p.ID == osPkgID || p.ID == bzlmodOsPkgID {
				osPkg = p
			}
		}

		if osPkg == nil {
			t.Errorf("Expected os package to be included:\n%+v", osPkg)
			return
		}

		if !osPkg.Standard {
			t.Errorf("Expected os import to be standard:\n%+v", osPkg)
			return
		}
	})
}

func TestExternalTests(t *testing.T) {
	resp := runForTest(t, "file=hello_external_test.go")
	if len(resp.Roots) != 2 {
		t.Errorf("Expected exactly two roots for package: %+v", resp.Roots)
	}

	var testId, xTestId string
	for _, id := range resp.Roots {
		if strings.HasSuffix(id, "_xtest") {
			xTestId = id
		} else {
			testId = id
		}
	}

	for _, p := range resp.Packages {
		if p.ID == xTestId {
			if !strings.HasSuffix(p.PkgPath, "_test") {
				t.Errorf("PkgPath missing _test suffix")
			}
			assertSuffixesInList(t, p.GoFiles, "/hello_external_test.go")
		} else if p.ID == testId {
			assertSuffixesInList(t, p.GoFiles, "/hello.go", "/hello_test.go")
		}
	}
}

func runForTest(t *testing.T, args ...string) driverResponse {
	t.Helper()

	// Remove most environment variables, other than those on an allowlist.
	//
	// Bazel sets TEST_* and RUNFILES_* and a bunch of other variables.
	// If Bazel is invoked when these variables, it assumes (correctly)
	// that it's being invoked by a test, and it does different things that
	// we don't want. For example, it randomizes the output directory, which
	// is extremely expensive here. Out test framework creates an output
	// directory shared among go_bazel_tests and points to it using .bazelrc.
	//
	// This only works if TEST_TMPDIR is not set when invoking bazel.
	// bazel_testing.BazelCmd normally unsets that, but since gopackagesdriver
	// invokes bazel directly, we need to unset it here.
	allowEnv := map[string]struct{}{
		"HOME":        {},
		"PATH":        {},
		"PWD":         {},
		"SYSTEMDRIVE": {},
		"SYSTEMROOT":  {},
		"TEMP":        {},
		"TMP":         {},
		"TZ":          {},
		"USER":        {},
	}
	var oldEnv []string
	for _, env := range os.Environ() {
		key, value, cut := strings.Cut(env, "=")
		if !cut {
			continue
		}
		if _, allowed := allowEnv[key]; !allowed {
			os.Unsetenv(key)
			oldEnv = append(oldEnv, key, value)
		}
	}
	defer func() {
		for i := 0; i < len(oldEnv); i += 2 {
			os.Setenv(oldEnv[i], oldEnv[i+1])
		}
	}()

	// Set workspaceRoot global variable.
	// It's initialized to the BUILD_WORKSPACE_DIRECTORY environment variable
	// before this point.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	oldWorkspaceRoot := workspaceRoot
	workspaceRoot = wd
	defer func() { workspaceRoot = oldWorkspaceRoot }()

	in := strings.NewReader("{}")
	out := &bytes.Buffer{}
	if err := run(context.Background(), in, out, args); err != nil {
		t.Fatalf("running gopackagesdriver: %v", err)
	}
	var resp driverResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	return resp
}

func assertSuffixesInList(t *testing.T, list []string, expectedSuffixes ...string) {
	t.Helper()
	for _, suffix := range expectedSuffixes {
		itemFound := false
		for _, listItem := range list {
			itemFound = itemFound || strings.HasSuffix(listItem, suffix)
		}

		if !itemFound {
			t.Errorf("Expected suffix %q in list, but was not found: %+v", suffix, list)
		}
	}
}
