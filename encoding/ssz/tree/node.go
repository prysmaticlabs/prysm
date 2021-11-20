package tree

import (
	"errors"
)

var zeroBytes = make([]byte, 32)

// Proof represents a merkle proof against a general index.
type Proof struct {
	Index  int
	Leaf   []byte
	Hashes [][]byte
}

// Node represents a node in the tree
// backing of a SSZ object.
type Node struct {
	left  *Node
	right *Node
	value []byte
}

// NewLeafWithValue initializes a leaf node.
func NewLeafWithValue(value []byte) *Node {
	return &Node{left: nil, right: nil, value: value}
}

// NewNodeWithLR initializes a branch node.
func NewNodeWithLR(left, right *Node) *Node {
	return &Node{left: left, right: right, value: nil}
}

// FromNodes constructs a tree from leaf nodes.
// This is useful for merging subtrees.
// The number of leaves should be a power of 2.
func FromNodes(leaves []*Node) (*Node, error) {
	numLeaves := len(leaves)

	if numLeaves == 1 {
		return leaves[0], nil
	}
	if numLeaves == 2 {
		return NewNodeWithLR(leaves[0], leaves[1]), nil
	}

	if !isPowerOfTwo(numLeaves) {
		return nil, errors.New("Number of leaves should be a power of 2")
	}

	numNodes := numLeaves*2 - 1
	nodes := make([]*Node, numNodes)
	for i := numNodes; i > 0; i-- {
		// Is a leaf
		if i > numNodes-numLeaves {
			nodes[i-1] = leaves[i-numLeaves]
		} else {
			// Is a branch node
			nodes[i-1] = &Node{left: nodes[(i*2)-1], right: nodes[(i*2+1)-1], value: nil}
		}
	}

	return nodes[0], nil
}

func FromNodesWithMixin(leaves []*Node, num, limit int) (*Node, error) {
	numLeaves := len(leaves)
	if !isPowerOfTwo(limit) {
		return nil, errors.New("Size of tree should be a power of 2")
	}

	allLeaves := make([]*Node, limit)
	emptyLeaf := NewLeafWithValue(make([]byte, 32))
	for i := 0; i < limit; i++ {
		if i < numLeaves {
			allLeaves[i] = leaves[i]
		} else {
			allLeaves[i] = emptyLeaf
		}
	}

	mainTree, err := FromNodes(allLeaves)
	if err != nil {
		return nil, err
	}

	// Mixin len
	countLeaf := LeafFromUint64(uint64(num))
	return NewNodeWithLR(mainTree, countLeaf), nil
}

// Get fetches a node in the tree with the given generalized index.
func (n *Node) Get(index int) (*Node, error) {
	pathLen := getPathLength(index)
	cur := n
	for i := pathLen - 1; i >= 0; i-- {
		if isRight := getPosAtLevel(index, i); isRight {
			cur = cur.right
		} else {
			cur = cur.left
		}
		if cur == nil {
			return nil, errors.New("Node not found in tree")
		}
	}

	return cur, nil
}

// Hash returns the hash of the subtree with the given Node as its root.
// If root has no children, it returns root's value (not its hash).
func (n *Node) Hash() []byte {
	return hashNode(n)
}

// Prove returns a list of sibling values and hashes needed
// to compute the root hash for a given general index.
// to compute the root hash for a given general index.
func (n *Node) Prove(index int) (*Proof, error) {
	pathLen := getPathLength(index)
	proof := &Proof{Index: index}
	hashes := make([][]byte, 0, pathLen)

	cur := n
	for i := pathLen - 1; i >= 0; i-- {
		var siblingHash []byte
		if isRight := getPosAtLevel(index, i); isRight {
			siblingHash = hashNode(cur.left)
			cur = cur.right
		} else {
			siblingHash = hashNode(cur.right)
			cur = cur.left
		}
		hashes = append([][]byte{siblingHash}, hashes...)
		if cur == nil {
			return nil, errors.New("Node not found in tree")
		}
	}

	proof.Hashes = hashes
	proof.Leaf = cur.value

	return proof, nil
}

func hashNode(n *Node) []byte {
	// Leaf
	if n.left == nil && n.right == nil {
		return n.value
	}
	// Only one child
	if n.left == nil || n.right == nil {
		panic("Tree incomplete")
	}
	return hashFn(append(hashNode(n.left), hashNode(n.right)...))
}

func isPowerOfTwo(n int) bool {
	return (n & (n - 1)) == 0
}
