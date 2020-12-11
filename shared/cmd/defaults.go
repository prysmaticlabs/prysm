// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := fileutil.HomeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Eth2")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Local", "Eth2")
		} else {
			return filepath.Join(home, ".eth2")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

// FixDefaultDataDir checks if previous data directory is found and can be migrated to a new path.
// This is used to resolve issue with weak default path (for Windows users) in existing installations.
// For full details see: https://github.com/prysmaticlabs/prysm/issues/5660.
func FixDefaultDataDir(prevDataDir, curDataDir string) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	// See if shared directory is found (if it is -- we need to move it to non-shared destination).
	prevDataDirExists, err := fileutil.HasDir(prevDataDir)
	if err != nil {
		return err
	}
	if !prevDataDirExists {
		// If no previous "%APPDATA%\Eth2" found, nothing to patch and move to new default location.
		return nil
	}

	if curDataDir == "" {
		curDataDir = DefaultDataDir()
	}
	selectedDirExists, err := fileutil.HasDir(curDataDir)
	if err != nil {
		return err
	}
	if selectedDirExists {
		// No need not move anything, destination directory already exists.
		return nil
	}

	if curDataDir == prevDataDir {
		return nil
	}

	log.Warnf("Previous data directory is found: %q. It is located in '%%APPDATA%%' and "+
		"needs to be relocated to a non-shared local folder: %q", prevDataDir, curDataDir)

	if err := os.Rename(prevDataDir, curDataDir); err != nil {
		return err
	}
	hasDir, err := fileutil.HasDir(curDataDir)
	fmt.Printf("After rename, hasDir: %v (err: %v)\n", hasDir, err)
	fmt.Printf("After rename, tosaccepted: %v\n", fileutil.FileExists(filepath.Join(curDataDir, "tosaccepted")))
	if err := os.Chmod(curDataDir, params.BeaconIoConfig().ReadWriteExecutePermissions); err != nil {
		return err
	}

	log.Infof("Data folder moved from %q to %q successfully!", prevDataDir, curDataDir)
	return nil
}
