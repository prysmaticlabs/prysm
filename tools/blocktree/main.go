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
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	params.UseDemoBeaconConfig()

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
		r, err := ssz.HashTreeRoot(b.Block)
		if err != nil {
			panic(err)
		}
		m[r] = &node{score: make(map[uint64]bool)}

		// Gather votes from the attestations voted for this block
		atts, err := db.Attestations(context.Background(), filters.NewFilter().SetHeadBlockRoot(r[:]))
		state, err := db.State(context.Background(), r)
		if err != nil {
			panic(err)
		}
		slot := b.Block.Slot
		// If the state is not available, roll back
		for state == nil {
			slot--
			filter := filters.NewFilter().SetStartSlot(slot).SetEndSlot(slot)
			bs, err := db.Blocks(context.Background(), filter)
			if err != nil {
				panic(err)
			}
			rs, err := ssz.HashTreeRoot(bs[0].Block)
			if err != nil {
				panic(err)
			}
			state, err = db.State(context.Background(), rs)
			if err != nil {
				panic(err)
			}
		}
		// Retrieve attestation indices
		for _, att := range atts {
			committee, err := helpers.BeaconCommitteeFromState(state, att.Data.Slot, att.Data.CommitteeIndex)
			if err != nil {
				panic(err)
			}
			indices, err := attestationutil.AttestingIndices(att.AggregationBits, committee)
			if err != nil {
				panic(err)
			}
			for _, i := range indices {
				m[r].score[i] = true
			}
		}

		// Construct label of each node.
		rStr := hex.EncodeToString(r[:2])
		label := "slot: " + strconv.Itoa(int(b.Block.Slot)) + "\n root: " + rStr + "\n votes: " + strconv.Itoa(len(m[r].score))

		dotN := graph.Node(rStr).Box().Attr("label", label)
		n := &node{
			parentRoot: bytesutil.ToBytes32(b.Block.ParentRoot),
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
