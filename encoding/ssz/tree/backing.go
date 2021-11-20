package tree

import (
	"math/bits"
)

type Backing struct {
	nodes []*Node
}

func (w *Backing) Indx() int {
	return len(w.nodes)
}

func (w *Backing) Node() *Node {
	if len(w.nodes) != 1 {
		panic("can only return if root node")
	}
	return w.nodes[0]
}

func (w *Backing) AddEmpty() {
	w.AddNode(EmptyLeaf())
}

func (w *Backing) AddNode(n *Node) {
	if w.nodes == nil {
		w.nodes = make([]*Node, 0)
	}
	w.nodes = append(w.nodes, n)
}

func (w *Backing) AddBytes(b []byte) {
	w.AddNode(LeafFromBytes(b))
}

func (w *Backing) AddUint64(i uint64) {
	w.AddNode(LeafFromUint64(i))
}

func (w *Backing) AddUint32(i uint32) {
	w.AddNode(LeafFromUint32(i))
}

func (w *Backing) AddUint16(i uint16) {
	w.AddNode(LeafFromUint16(i))
}

func (w *Backing) AddUint8(i uint8) {
	w.AddNode(LeafFromUint8(i))
}

func (w *Backing) AddBitlist(blist []byte, maxSize int) {
	tmp, size := parseBitlistForTree(blist)
	subIdx := w.Indx()
	w.AddBytes(tmp)
	w.CommitWithMixin(subIdx, int(size), (maxSize+255)/256)
}

func (w *Backing) Commit(i int) {
	res, err := FromNodes(w.nodes[i:])
	if err != nil {
		panic(err)
	}
	// remove the old nodes
	w.nodes = w.nodes[:i]
	// add the new node
	w.AddNode(res)
}

func (w *Backing) CommitWithMixin(i, num, limit int) {
	res, err := FromNodesWithMixin(w.nodes[i:], num, limit)
	if err != nil {
		panic(err)
	}
	// remove the old nodes
	w.nodes = w.nodes[:i]
	// add the new node
	w.AddNode(res)
}

func parseBitlistForTree(buf []byte) ([]byte, uint64) {
	dst := make([]byte, 0)
	msb := uint8(bits.Len8(buf[len(buf)-1])) - 1
	size := uint64(8*(len(buf)-1) + int(msb))

	dst = append(dst, buf...)
	dst[len(dst)-1] &^= uint8(1 << msb)

	newLen := len(dst)
	for i := len(dst) - 1; i >= 0; i-- {
		if dst[i] != 0x00 {
			break
		}
		newLen = i
	}
	res := dst[:newLen]
	return res, size
}
