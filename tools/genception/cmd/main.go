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
	"os/signal"
	"strings"

	"github.com/prysmaticlabs/prysm/v5/tools/genception/driver"
)

var log = driver.Logger

func run(_ context.Context, in io.Reader, out io.Writer, args []string) error {
	rec, err := driver.NewRecorder()
	if err != nil {
		return fmt.Errorf("unable to initialize recorder: %w", err)
	}
	resolver, err := driver.NewPathResolver()
	if err != nil {
		return fmt.Errorf("unable to initialize path resolver: %w", err)
	}
	jsonFiles, err := driver.LoadJsonListing()
	if err != nil {
		return fmt.Errorf("unable to lookup package: %w", err)
	}
	pd, err := driver.NewJSONPackagesDriver(jsonFiles, resolver.Resolve)
	if err != nil {
		return fmt.Errorf("unable to load JSON files: %w", err)
	}

	request, err := driver.ReadDriverRequest(in)
	if err != nil {
		return fmt.Errorf("unable to read request: %w", err)
	}
	if err := rec.RecordRequest(args, request); err != nil {
		return fmt.Errorf("unable to record request: %w", err)
	}
	// Note: we are returning all files required to build a specific package.
	// For file queries (`file=`), this means that the CompiledGoFiles will
	// include more than the only file being specified.
	resp := pd.Handle(request, args)
	if err := rec.RecordResponse(resp); err != nil {
		return fmt.Errorf("unable to record response: %w", err)
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("unable to marshal response: %v", err)
	}
	_, err = out.Write(data)
	return err
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.WithField("args", strings.Join(os.Args[1:], " ")).Info("genception lookup")
	if err := run(ctx, os.Stdin, os.Stdout, os.Args[1:]); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "error: %v", err)
		if err != nil {
			log.WithError(err).Error("unhandled error in package resolution")
		}
		// gopls will check the packages driver exit code, and if there is an
		// error, it will fall back to go list. Obviously we don't want that,
		// so force a 0 exit code.
		os.Exit(0)
	}
}
