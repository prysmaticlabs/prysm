// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.
package fileutil_test

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestPathExpansion(t *testing.T) {
	user, err := user.Current()
	require.NoError(t, err)
	tests := map[string]string{
		"/home/someuser/tmp": "/home/someuser/tmp",
		"~/tmp":              user.HomeDir + "/tmp",
		"$DDDXXX/a/b":        "/tmp/a/b",
		"/a/b/":              "/a/b",
	}
	require.NoError(t, os.Setenv("DDDXXX", "/tmp"))
	for test, expected := range tests {
		expanded, err := fileutil.ExpandPath(test)
		require.NoError(t, err)
		assert.Equal(t, expected, expanded)
	}
}

func TestMkdirAll_AlreadyExists_WrongPermissions(t *testing.T) {
	dirName := t.TempDir() + "somedir"
	err := os.MkdirAll(dirName, os.ModePerm)
	require.NoError(t, err)
	err = fileutil.MkdirAll(dirName)
	assert.ErrorContains(t, "already exists without proper 0700 permissions", err)
}

func TestMkdirAll_AlreadyExists_OK(t *testing.T) {
	dirName := t.TempDir() + "somedir"
	err := os.MkdirAll(dirName, params.BeaconIoConfig().ReadWriteExecutePermissions)
	require.NoError(t, err)
	assert.NoError(t, fileutil.MkdirAll(dirName))
}

func TestMkdirAll_OK(t *testing.T) {
	dirName := t.TempDir() + "somedir"
	err := fileutil.MkdirAll(dirName)
	assert.NoError(t, err)
	exists, err := fileutil.HasDir(dirName)
	require.NoError(t, err)
	assert.Equal(t, true, exists)
}

func TestWriteFile_AlreadyExists_WrongPermissions(t *testing.T) {
	dirName := t.TempDir() + "somedir"
	err := os.MkdirAll(dirName, os.ModePerm)
	require.NoError(t, err)
	someFileName := filepath.Join(dirName, "somefile.txt")
	require.NoError(t, ioutil.WriteFile(someFileName, []byte("hi"), os.ModePerm))
	err = fileutil.WriteFile(someFileName, []byte("hi"))
	assert.ErrorContains(t, "already exists without proper 0600 permissions", err)
}

func TestWriteFile_AlreadyExists_OK(t *testing.T) {
	dirName := t.TempDir() + "somedir"
	err := os.MkdirAll(dirName, os.ModePerm)
	require.NoError(t, err)
	someFileName := filepath.Join(dirName, "somefile.txt")
	require.NoError(t, ioutil.WriteFile(someFileName, []byte("hi"), params.BeaconIoConfig().ReadWritePermissions))
	assert.NoError(t, fileutil.WriteFile(someFileName, []byte("hi")))
}

func TestWriteFile_OK(t *testing.T) {
	dirName := t.TempDir() + "somedir"
	err := os.MkdirAll(dirName, os.ModePerm)
	require.NoError(t, err)
	someFileName := filepath.Join(dirName, "somefile.txt")
	require.NoError(t, fileutil.WriteFile(someFileName, []byte("hi")))
	exists := fileutil.FileExists(someFileName)
	assert.Equal(t, true, exists)
}

func TestCopyFile(t *testing.T) {
	fName := t.TempDir() + "testfile"
	err := ioutil.WriteFile(fName, []byte{1, 2, 3}, params.BeaconIoConfig().ReadWritePermissions)
	require.NoError(t, err)

	err = fileutil.CopyFile(fName, fName+"copy")
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, os.Remove(fName+"copy"))
	}()

	assert.Equal(t, true, deepCompare(t, fName, fName+"copy"))
}

func TestCopyDir(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := filepath.Join(t.TempDir(), "copyfolder")
	type fileDesc struct {
		path    string
		content []byte
	}
	fds := []fileDesc{
		{
			path:    "testfile1",
			content: []byte{1, 2, 3},
		},
		{
			path:    "subfolder1/testfile1",
			content: []byte{4, 5, 6},
		},
		{
			path:    "subfolder1/testfile2",
			content: []byte{7, 8, 9},
		},
		{
			path:    "subfolder2/testfile1",
			content: []byte{10, 11, 12},
		},
		{
			path:    "testfile2",
			content: []byte{13, 14, 15},
		},
	}
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir1, "subfolder1"), 0777))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir1, "subfolder2"), 0777))
	for _, fd := range fds {
		require.NoError(t, fileutil.WriteFile(filepath.Join(tmpDir1, fd.path), fd.content))
		assert.Equal(t, true, fileutil.FileExists(filepath.Join(tmpDir1, fd.path)))
		assert.Equal(t, false, fileutil.FileExists(filepath.Join(tmpDir2, fd.path)))
	}

	// Make sure that files are copied into non-existent directory only. If directory exists function exits.
	assert.ErrorContains(t, "destination directory already exists", fileutil.CopyDir(tmpDir1, t.TempDir()))
	require.NoError(t, fileutil.CopyDir(tmpDir1, tmpDir2))

	// Now, all files should have been copied.
	for _, fd := range fds {
		assert.Equal(t, true, fileutil.FileExists(filepath.Join(tmpDir2, fd.path)))
		assert.Equal(t, true, deepCompare(t, filepath.Join(tmpDir1, fd.path), filepath.Join(tmpDir2, fd.path)))
	}
}

func TestDirFiles(t *testing.T) {
	readDir := func(dir string) []string {
		fds, err := ioutil.ReadDir(dir)
		require.NoError(t, err)
		var fnames []string
		for _, fd := range fds {
			if !fd.IsDir() {
				fnames = append(fnames, fd.Name())
			}
		}
		return fnames
	}
	tmpDir, tmpDirFnames := tmpDirWithContents(t)
	tests := []struct {
		name     string
		path     string
		outFiles []string
	}{
		{
			name:     "empty path",
			path:     "",
			outFiles: readDir("."),
		},
		{
			name:     "dot path",
			path:     ".",
			outFiles: readDir("."),
		},
		{
			name:     "non-empty folder",
			path:     tmpDir,
			outFiles: tmpDirFnames,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFiles, err := fileutil.DirFiles(tt.path)
			require.NoError(t, err)

			sort.Strings(outFiles)
			assert.DeepEqual(t, tt.outFiles, outFiles)
		})
	}
}

func deepCompare(t *testing.T, file1, file2 string) bool {
	sf, err := os.Open(file1)
	assert.NoError(t, err)
	df, err := os.Open(file2)
	assert.NoError(t, err)
	sscan := bufio.NewScanner(sf)
	dscan := bufio.NewScanner(df)

	for sscan.Scan() && dscan.Scan() {
		if !bytes.Equal(sscan.Bytes(), dscan.Bytes()) {
			return false
		}
	}
	return true
}

// tmpDirWithContents returns path to temporary directory having some folders/files in it.
// Directory is automatically removed by internal testing cleanup methods.
func tmpDirWithContents(t *testing.T) (string, []string) {
	dir := t.TempDir()
	fnames := []string{
		"file1",
		"file2",
		"subfolder1/file1",
		"subfolder1/file2",
		"subfolder1/subfolder11/file1",
		"subfolder1/subfolder11/file2",
		"subfolder1/subfolder12/file1",
		"subfolder1/subfolder12/file2",
		"subfolder2/file1",
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subfolder1", "subfolder11"), 0777))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subfolder1", "subfolder12"), 0777))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subfolder2"), 0777))
	for _, fname := range fnames {
		require.NoError(t, ioutil.WriteFile(filepath.Join(dir, fname), []byte(fname), 0777))
	}
	sort.Strings(fnames)
	return dir, fnames
}
