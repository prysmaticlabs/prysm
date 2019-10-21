package dag

import (
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// Dag
type Dag struct {
	// Nodes
	Nodes map[[32]byte]*Node
	// Scores
	Scores map[*Node]uint64
	// Finalized
	Finalized *Node
	// Justified
	Justified *Node
	synced bool
	maxKnownSlot uint64
}

// New
func New() *Dag {
	return &Dag{
		Nodes: make(map[[32]byte]*Node),
		Scores: make(map[*Node]uint64),
		synced: false,
	}
}

// AddNode
func (d *Dag) AddNode(block *ethpb.BeaconBlock) error {
	d.synced = false
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		return err
	}
	node := &Node{
		Parent: d.Nodes[bytesutil.ToBytes32(block.ParentRoot)],
		Children: make([]*Node, 0, 8),
		Slot: block.Slot,
		Weight: 0,
		Key: blockRoot,
	}

	node.IndexAsChild = uint64(len(node.Parent.Children))
	node.Parent.Children = append(node.Parent.Children, node)

	d.Nodes[blockRoot] = node

	if d.Finalized == nil {
		d.Finalized = node
	}
	if d.Justified == nil {
		d.Justified = node
	}

	d.maxKnownSlot = block.Slot

	return nil
}

// ApplyScoreChanges
func (d *Dag) ApplyScoreChanges(changes []ScoreChange) {
	for _, v := range changes {
		if v.Target.Slot >= d.Finalized.Slot {
			d.Scores[v.Target] += v.Delta
		}
	}
	// delete targets that have a 0 score
	for k, v := range d.Scores {
		if v == 0 {
			delete(d.Scores, k)
		}
	}
}

// Head
func (d *Dag) Head() *Node {
	start := d.Justified
	// Track weight for each block per height.
	weightAtHeight := make([]map[*Node]uint64, d.maxKnownSlot + 1 - start.Slot)
	for i := 0; i < len(weightAtHeight); i++ {
		weightAtHeight[i] = make(map[*Node]uint64)
	}

	// Compute cutoff to stop and return head.
	cutoff := uint64(0)
	for n, s := range d.Scores {
		if n.Slot > start.Slot {
			weightAtHeight[n.Slot - start.Slot][n] += s
			cutoff += s
		}
	}
	cutoff /= 2

	bestChild := make(map[*Node]*ChildScore)
	// Back propagate highest slot weights back to root of the tree,
	// Also track the most weighted child.
	for i:=d.maxKnownSlot - start.Slot; i > 0; i-- {
		for n, w := range weightAtHeight[i] {
			if w > cutoff {
				if best, hasBest := bestChild[n]; hasBest {
					return best.BestTarget
				} else {
					return n
				}
			}
			// Propagate the weight of child to parent.
			weightAtHeight[n.Parent.Slot - start.Slot][n.Parent] = weightAtHeight[n.Parent.Slot - start.Slot][n.Parent] + w
			// Track the best child for parent block.
			children, has := bestChild[n.Parent]
			if !has || w > children.Score {
				if best, hasBest := bestChild[n]; hasBest {
					// inherit the best-target if there is one.
					bestChild[n.Parent] = &ChildScore{BestTarget: best.BestTarget, Score: w}
				} else {
					// otherwise just put this node as the best target.
					bestChild[n.Parent] = &ChildScore{BestTarget: n, Score: w}
				}
			}
		}
	}

	// Worst case scenario, process head at justified check point height.
	if myBest, hasBest := bestChild[start]; hasBest {
		return myBest.BestTarget
	} else {
		return d.Justified
	}
}
