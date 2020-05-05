package main

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/urfave/cli.v2"
)

func TestAllFlagsExistInHelp(t *testing.T) {
	// If this test is failing, it is because you've recently added/removed a
	// flag in beacon chain main.go, but did not add/remove it to the usage.go
	// flag grouping (appHelpFlagGroups).

	var helpFlags []cli.Flag
	for _, group := range appHelpFlagGroups {
		helpFlags = append(helpFlags, group.Flags...)
	}
	helpFlags = featureconfig.ActiveFlags(helpFlags)
	appFlags = featureconfig.ActiveFlags(appFlags)

	for _, flag := range appFlags {
		if !doesFlagExist(flag, helpFlags) {
			t.Errorf("Flag %s does not exist in help/usage flags.", flag.Names()[0])
		}
	}

	for _, flag := range helpFlags {
		if !doesFlagExist(flag, appFlags) {
			t.Errorf("Flag %s does not exist in main.go, "+
				"but exists in help flags", flag.Names()[0])
		}
	}
}

func doesFlagExist(flag cli.Flag, flags []cli.Flag) bool {
	for _, f := range flags {
		if f.String() == flag.String() {
			return true
		}
	}

	return false
}

func TestLoadConfigFile(t *testing.T) {
	mainnetConfigFile := testutil.ConfigFilePath(t, "mainnet")
	loadChainConfigFile(mainnetConfigFile)
	if params.BeaconConfig().MaxCommitteesPerSlot != params.MainnetConfig().MaxCommitteesPerSlot {
		t.Errorf("Expected MaxCommitteesPerSlot to be set to mainnet value: %d found: %d",
			params.MainnetConfig().MaxCommitteesPerSlot,
			params.BeaconConfig().MaxCommitteesPerSlot)
	}
	if params.BeaconConfig().SecondsPerSlot != params.MainnetConfig().SecondsPerSlot {
		t.Errorf("Expected SecondsPerSlot to be set to mainnet value: %d found: %d",
			params.MainnetConfig().SecondsPerSlot,
			params.BeaconConfig().SecondsPerSlot)
	}
	minimalConfigFile := testutil.ConfigFilePath(t, "minimal")
	loadChainConfigFile(minimalConfigFile)
	if params.BeaconConfig().MaxCommitteesPerSlot != params.MinimalSpecConfig().MaxCommitteesPerSlot {
		t.Errorf("Expected MaxCommitteesPerSlot to be set to minimal value: %d found: %d",
			params.MinimalSpecConfig().MaxCommitteesPerSlot,
			params.BeaconConfig().MaxCommitteesPerSlot)
	}
	if params.BeaconConfig().SecondsPerSlot != params.MinimalSpecConfig().SecondsPerSlot {
		t.Errorf("Expected SecondsPerSlot to be set to minimal value: %d found: %d",
			params.MinimalSpecConfig().SecondsPerSlot,
			params.BeaconConfig().SecondsPerSlot)
	}
}
