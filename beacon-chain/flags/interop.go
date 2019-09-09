package flags

import (
	"github.com/urfave/cli"
)

var (
	// InteropGenesisState defines a flag for the beacon node to load genesis state via file.
	InteropGenesisState = cli.StringFlag{
		Name:  "interop-genesis-state",
		Usage: "The genesis state file (.SSZ) to load from",
	}

	// "Cold start" flags to use a deterministic genesis state generation.
	//
	InteropGenesisTime = cli.Uint64Flag{
		Name: "interop-genesis-time",
		Usage: "",
	}
	//
	InteropNumValidators = cli.Uint64Flag{
		Name: "interop-num-validators",
		Usage: "",
	}
)
