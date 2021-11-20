package tree

import "encoding/binary"

func EmptyLeaf() *Node {
	return NewLeafWithValue(zeroBytes[:32])
}

func LeafFromUint64(i uint64) *Node {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf[:8], i)
	return NewLeafWithValue(buf)
}

func LeafFromUint32(i uint32) *Node {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint32(buf[:4], i)
	return NewLeafWithValue(buf)
}

func LeafFromUint16(i uint16) *Node {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint16(buf[:2], i)
	return NewLeafWithValue(buf)
}

func LeafFromUint8(i uint8) *Node {
	buf := make([]byte, 32)
	buf[0] = i
	return NewLeafWithValue(buf)
}

func LeafFromBytes(b []byte) *Node {
	l := len(b)
	if l > 32 {
		panic("Unimplemented")
	}

	if l == 32 {
		return NewLeafWithValue(b[:])
	}
	return NewLeafWithValue(append(b, zeroBytes[:32-l]...))
}

func LeavesFromBytes(items [][]byte) []*Node {
	if len(items) == 0 {
		return []*Node{}
	}

	numLeaves := (len(items)*8 + 31) / 32
	leaves := make([]*Node, numLeaves)
	for i := 0; i < numLeaves; i++ {
		leaves[i] = NewLeafWithValue(items[i])
	}

	return leaves
}

func LeavesFromUint64(items []uint64) []*Node {
	if len(items) == 0 {
		return []*Node{}
	}

	numLeaves := (len(items)*8 + 31) / 32
	buf := make([]byte, numLeaves*32)
	for i, v := range items {
		binary.LittleEndian.PutUint64(buf[i*8:(i+1)*8], v)
	}

	leaves := make([]*Node, numLeaves)
	for i := 0; i < numLeaves; i++ {
		v := buf[i*32 : (i+1)*32]
		leaves[i] = NewLeafWithValue(v)
	}

	return leaves
}
