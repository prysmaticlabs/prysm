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

package cmd

import (
	"os"
	"os/user"
	"testing"
)

var (
  // user-specific tests can use the usr variable.
  usr, _ = user.Current()
  against []DirectoryString
)

func init() {
  // test for environment variables in paths with $DDDXXX.
  os.Setenv("DDDXXX", "/tmp")
  // test against valid DirectoryString structs.
  against = []DirectoryString{
	  {"/home/someuser/tmp"},
	  {usr.HomeDir + "/tmp"},
	  {"~thisOtherUser/b"},
    {"/tmp/a/b"},
    {"/a/b"},
  }
}

func TestPathExpansion(t *testing.T) {
  initials := []string{
		"/home/someuser/tmp",
		"~/tmp",
		"~thisOtherUser/b/",
		"$DDDXXX/a/b",
		"/a/b/",
  }
  tests := make(map[string]DirectoryString)
  for i, value := range against {
    tests[initials[i]] = value
  }
	for test, expected := range tests {
		got := expandPath(test)
		if got != expected.Value {
			t.Errorf("test path expansion %s, got %s, expected %s\n", test, got, expected)
		}
	}
}

func TestSetDirectoryString(t *testing.T) {
  tests := make([]DirectoryString, len(against))
  copy(tests, against)
  expected := "/tmp"
  for _, test := range tests {
    original := test.Value
    test.Set("$DDDXXX")
    got := test.Value
    if test.Value != expected {
      t.Errorf("test set path %s, got %s, expected %s\n", original, got, expected)
    }
  }
}

func TestPrefixFor(t *testing.T) {
  tests := map[string]string{
    "d": "-",
    "datadir": "--",
    "": "--",
  }

  for test, expected := range tests {
    actual := prefixFor(test)
    if actual != expected {
      t.Errorf("test prefix for %s, got %s, expected %s\n", test, actual, expected)
    }
  }
}

func TestPrefixedNames(t *testing.T) {
  test := "d,datadir,,n,network"
  expected := "-d, --datadir, --, -n, --network"

  prefixed := prefixedNames(test)

  if prefixed != expected {
    t.Errorf("test prefixed names for %s, got %s, expected %s\n", test, prefixed, expected)
  }
}

func TestHomeDir(t *testing.T) {
  expected := usr.HomeDir
  got := homeDir()
  if got != expected {
    t.Errorf("test homeDir(): got %s, expected %s\n", got, expected)
  }
}
