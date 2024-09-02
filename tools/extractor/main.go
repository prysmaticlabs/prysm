package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db/kv"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v5/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	"github.com/prysmaticlabs/prysm/v5/config/params"
)

var (
	// Required fields
	datadir = flag.String("datadir", "", "Path to data directory.")
)

func main() {
	resetCfg := features.InitWithReset(&features.Flags{WriteSSZStateTransitions: true})
	defer resetCfg()
	flag.Parse()
	fmt.Println("Starting process...")
	d, err := kv.NewKVStore(context.Background(), *datadir)
	if err != nil {
		panic(err)
	}

	cfg := params.BeaconConfig()
	cfg.ChurnLimitQuotient = 128 // Devnet uses this config. Without it, state transition will fail when processing epoch
	params.OverrideBeaconConfig(cfg)

	ctx := context.Background()

	fc := doublylinkedtree.New()
	s := stategen.New(d, fc)
	hexString := "04caf4ff7fc7ab2b294ae83ea7a4c1f1763a3e552a08735a6a5d5755b4fd4933" // This is the finalized root, which we'll first replay to and save it as finalized root for state transition
	root, err := hex.DecodeString(hexString)
	if err != nil {
		panic(err)
	}
	st, err := s.StateByRoot(ctx, [32]byte(root))
	if err != nil {
		panic(err)
	}
	_, err = s.Resume(ctx, st) // Resume saves the caches of the finalized state for state gen
	if err != nil {
		panic(err)
	}

	hexString = "ce3665faa64345557b47afa03ef7b81e48c24c6a712e4c3c90303cef64e88af8" // This is the target root
	root, err = hex.DecodeString(hexString)
	if err != nil {
		panic(err)
	}
	st, err = s.StateByRoot(ctx, [32]byte(root))
	if err != nil {
		panic(err)
	}

	st, err = s.StateByRoot(ctx, [32]byte(root)) // Replay twice fails because start state caches are incorrect
	if err != nil {
		// Error: panic: state root 0xd21dac2ac7a03561f17c550bcaa7d5d5be8b9f5cbd2e23ae808fc8306e290be9
		// does not match the block state root 0x3952f2c8a6eebc7ff96f05ca54fca81363a0fb720238722e00197a7c1e3c7149
		panic(err)
	}
}
