package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

var (
	// Required fields
	datadir = flag.String("datadir", "", "Path to data directory.")

	state = flag.Uint("state", 0, "Extract state at this slot.")
)

func init() {
	fc := featureconfig.Get()
	fc.WriteSSZStateTransitions = true
	featureconfig.Init(fc)
}

func main() {
	flag.Parse()
	fmt.Println("Starting process...")
	d, err := db.NewDB(*datadir)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	slot := uint64(*state)
	roots, err := d.BlockRoots(ctx, filters.NewFilter().SetStartSlot(slot).SetEndSlot(slot))
	if err != nil {
		panic(err)
	}
	if len(roots) != 1 {
		fmt.Printf("Expected 1 block root for slot %d, got %d roots", *state, len(roots))
	}
	s, err := d.State(ctx, roots[0])
	if err != nil {
		panic(err)
	}

	interop.WriteStateToDisk(s)
	fmt.Println("done")
}
