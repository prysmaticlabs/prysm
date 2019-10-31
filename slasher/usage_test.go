package main

import (
	"testing"

	"github.com/urfave/cli"
)

func TestAllFlagsExistInHelp(t *testing.T) {
	// If this test is failing, it is because you've recently added/removed a
	// flag in beacon chain main.go, but did not add/remove it to the usage.go
	// flag grouping (appHelpFlagGroups).

	var helpFlags []cli.Flag
	for _, group := range appHelpFlagGroups {
		helpFlags = append(helpFlags, group.Flags...)
	}

	for _, flag := range appFlags {
		if !doesFlagExist(flag, helpFlags) {
			t.Errorf("Flag %s does not exist in help/usage flags.", flag.GetName())
		}
	}

	for _, flag := range helpFlags {
		if !doesFlagExist(flag, appFlags) {
			t.Errorf("Flag %s does not exist in main.go, "+
				"but exists in help flags", flag.GetName())
		}
	}
}

func doesFlagExist(flag cli.Flag, flags []cli.Flag) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}

	return false
}
