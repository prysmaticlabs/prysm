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
	"fmt"
	"go/build"
	"os"
	"os/signal"
	"path"
	"path/filepath"
)

func getenvDefault(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultValue
}

func concatStringsArrays(values ...[]string) []string {
	ret := []string{}
	for _, v := range values {
		ret = append(ret, v...)
	}
	return ret
}

func ensureAbsolutePathFromWorkspace(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(workspaceRoot, path)
}

func signalContext(parentCtx context.Context, signals ...os.Signal) (ctx context.Context, stop context.CancelFunc) {
	ctx, cancel := context.WithCancel(parentCtx)
	ch := make(chan os.Signal, 1)
	go func() {
		select {
		case <-ch:
			cancel()
		case <-ctx.Done():
		}
	}()
	signal.Notify(ch, signals...)

	return ctx, cancel
}

func isLocalPattern(pattern string) bool {
	return build.IsLocalImport(pattern) || filepath.IsAbs(pattern)
}

func packageID(pattern string) string {
	pattern = path.Clean(pattern)
	if filepath.IsAbs(pattern) {
		if relPath, err := filepath.Rel(workspaceRoot, pattern); err == nil {
			pattern = relPath
		}
	}

	return fmt.Sprintf("//%s", pattern)
}
