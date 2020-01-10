/**
 * Genesis state differ between two beacon nodes
 *
 * Given two beacon nodes' DB's, this tool looks at the differences between their
 * genesis states, genesis blocks, and genesis state roots for debugging.
 */
package main

import (
	"context"
	"flag"
	"log"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/stateutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

var (
	datadir1 = flag.String("a", "", "Path to node A's datadir")
	datadir2 = flag.String("b", "", "Path to node B's datadir")
)

func main() {
	params.UseDemoBeaconConfig()

	flag.Parse()
	db1, err := db.NewDB(*datadir1)
	if err != nil {
		log.Fatal(err)
	}
	db2, err := db.NewDB(*datadir2)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	s1, err := db1.GenesisState(ctx)
	if err != nil {
		log.Fatal(err)
	}
	stateRoot1, err := stateutil.HashTreeRootState(s1)
	if err != nil {
		log.Fatal(err)
	}
	s2, err := db2.GenesisState(ctx)
	if err != nil {
		log.Fatal(err)
	}
	stateRoot2, err := stateutil.HashTreeRootState(s2)
	if err != nil {
		log.Fatal(err)
	}

	gs1, err := db1.GenesisBlock(ctx)
	if err != nil {
		log.Fatal(err)
	}
	gs2, err := db2.GenesisBlock(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if !proto.Equal(s1, s2) {
		log.Println(messagediff.PrettyDiff(s1, s2))
	} else {
		log.Println("States match")
	}

	if stateRoot1 != stateRoot2 {
		log.Printf("State roots mismatch, want %#x got %#x", stateRoot1, stateRoot2)
	} else {
		log.Printf("Genesis state roots match, %#x == %#x", stateRoot1, stateRoot2)
	}

	if !proto.Equal(gs1, gs2) {
		log.Printf("Genesis block state root mismatch, %v", messagediff.PrettyDiff(gs1, gs2))
	} else {
		log.Println("Genesis blocks match")
	}
}
