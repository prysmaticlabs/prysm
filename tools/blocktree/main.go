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
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
}

func main() {
	flag.Parse()
	db, err := db.NewDB(*datadir)
	if err != nil {
		panic(err)
	}

	graph := dot.NewGraph(dot.Directed)
	graph.Attr("rankdir", "RL")
	graph.Attr("labeljust", "l")

	startSlot := uint64(*startSlot)
	endSlot := uint64(*endSlot)
	filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot)
	blks, err := db.Blocks(context.Background(), filter)
	if err != nil {
		panic(err)
	}

	// Construct nodes
	m := make(map[[32]byte]*node)
	for i := 0; i < len(blks); i++ {
		b := blks[i]
		r, err := ssz.SigningRoot(b)
		if err != nil {
			panic(err)
		}
		// Construct label of each node.
		rStr := hex.EncodeToString(r[:2])
		label := "slot: " + strconv.Itoa(int(b.Slot)) + "\n root: " + rStr
		dotN := graph.Node(rStr).Box().Attr("label", label)
		n := &node{
			parentRoot: bytesutil.ToBytes32(b.ParentRoot),
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
