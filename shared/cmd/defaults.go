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
	"os"
	"path/filepath"
	"runtime"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
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

// CheckDefaultDataDir checks and fixes the issue with default data directory path.
// For full details see: https://github.com/prysmaticlabs/prysm/issues/5660.
func CheckDefaultDataDir(selectedDir string) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	// See if shared directory is found (if it is -- we need to move it to non-shared destination).
	roamingAppDataDir := filepath.Join(fileutil.HomeDir(), "AppData", "Roaming", "Eth2")
	roamingAppDataDirExists, err := fileutil.HasDir(roamingAppDataDir)
	if err != nil {
		return err
	}
	if !roamingAppDataDirExists {
		// If no previous "%APPDATA%\Eth2" found, nothing to patch and move to new default location.
		return nil
	}

	if selectedDir == "" {
		selectedDir = DefaultDataDir()
	}
	selectedDirExists, err := fileutil.HasDir(selectedDir)
	if err != nil {
		return err
	}
	if selectedDirExists {
		// No need not move anything, destination directory already exists.
		return nil
	}

	if selectedDir == roamingAppDataDir {
		return nil
	}

	log.Warnf("Previous data directory is found: %q. It is located in '%%APPDATA%%' and "+
		"needs to be relocated to a non-shared local folder: %q", roamingAppDataDir, selectedDir)

	if err := os.Rename(roamingAppDataDir, selectedDir); err != nil {
		return err
	}

	log.Infof("Data folder moved from %q to %q successfully!", roamingAppDataDir, selectedDir)
	return nil
}
