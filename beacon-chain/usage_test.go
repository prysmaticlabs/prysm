package main

import (
	"testing"

	"github.com/urfave/cli"
)

func TestAllFlagsExistInHelp(t *testing.T) {
	var helpFlags []cli.Flag
	for _, group := range appHelpFlagGroups {
		helpFlags = append(helpFlags, group.Flags...)
	}

	for _, flag := range appFlags {
		if !doesFlagExist(flag, helpFlags) {
			t.Errorf("Flag %s does not exist in help/usage flags.", flag.GetName())
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
