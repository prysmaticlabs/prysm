/**
 * Block tree graph viz
 *
 * Given a DB, start slot and end slot. This tool computes the graphviz data
 * needed to construct the block tree in graphviz data format. Then one can paste
 * the data in a Graph rendering engine (ie. http://www.webgraphviz.com/) to see the visual format.

 */
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"strconv"

	"github.com/emicklei/dot"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

var (
	// Required fields
	datadir   = flag.String("datadir", "", "Path to data directory.")
	startSlot = flag.Uint("startSlot", 0, "Start slot of the block tree")
	endSlot   = flag.Uint("endSlot", 0, "Start slot of the block tree")
)

// Used for tree, each node is a representation of a node in the graph
type node struct {
	parentRoot [32]byte
	dothNode   *dot.Node
	score      map[uint64]bool
}

func main() {
	flag.Parse()
	db, err := db.NewDB(context.Background(), *datadir, &kv.Config{})
	if err != nil {
		panic(err)
	}

	graph := dot.NewGraph(dot.Directed)
	graph.Attr("rankdir", "RL")
	graph.Attr("labeljust", "l")

	startSlot := types.Slot(*startSlot)
	endSlot := types.Slot(*endSlot)
	filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot)
	blks, roots, err := db.Blocks(context.Background(), filter)
	if err != nil {
		panic(err)
	}

	// Construct nodes
	m := make(map[[32]byte]*node)
	for i := 0; i < len(blks); i++ {
		b := blks[i]
		r := roots[i]
		m[r] = &node{score: make(map[uint64]bool)}

		state, err := db.State(context.Background(), r)
		if err != nil {
			panic(err)
		}
		slot := b.Block().Slot()
		// If the state is not available, roll back
		for state == nil {
			slot--
			_, rts, err := db.BlockRootsBySlot(context.Background(), slot)
			if err != nil {
				panic(err)
			}
			state, err = db.State(context.Background(), rts[0])
			if err != nil {
				panic(err)
			}
		}

		// Construct label of each node.
		rStr := hex.EncodeToString(r[:2])
		label := "slot: " + strconv.Itoa(int(b.Block().Slot())) + "\n root: " + rStr

		dotN := graph.Node(rStr).Box().Attr("label", label)
		n := &node{
			parentRoot: bytesutil.ToBytes32(b.Block().ParentRoot()),
			dothNode:   &dotN,
		}
		m[r] = n
	}

	// Construct an edge only if block's parent exist in the tree.
	for _, n := range m {
		if _, ok := m[n.parentRoot]; ok {
			graph.Edge(*n.dothNode, *m[n.parentRoot].dothNode)
		}
	}

	fmt.Println(graph.String())
}
