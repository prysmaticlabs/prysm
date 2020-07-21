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

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestPathExpansion(t *testing.T) {
	user, err := user.Current()
	if err != nil {
		t.Error(err)
	}
	tests := map[string]string{
		"/home/someuser/tmp": "/home/someuser/tmp",
		"~/tmp":              user.HomeDir + "/tmp",
		"~thisOtherUser/b/":  "~thisOtherUser/b",
		"$DDDXXX/a/b":        "/tmp/a/b",
		"/a/b/":              "/a/b",
	}
	err = os.Setenv("DDDXXX", "/tmp")
	if err != nil {
		t.Error(err)
	}
	for test, expected := range tests {
		assert.Equal(t, expected, expandPath(test))
	}
}
