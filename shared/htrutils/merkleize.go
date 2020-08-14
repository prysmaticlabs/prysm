// Package htrutils defines HashTreeRoot utility functions.
package htrutils

import (
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// Merkleize.go is mostly a directly copy of the same filename from
// 	https://github.com/protolambda/zssz/blob/master/merkle/merkleize.go.
// The reason the method is copied instead of imported is due to us using a
// a custom hasher interface for a reduced memory footprint when using
// 'Merkleize'.

const (
	mask0 = ^uint64((1 << (1 << iota)) - 1)
	mask1
	mask2
	mask3
	mask4
	mask5
)

const (
	bit0 = uint8(1 << iota)
	bit1
	bit2
	bit3
	bit4
	bit5
)

// GetDepth retrieves the appropriate depth for the provided trie size.
func GetDepth(v uint64) (out uint8) {
	// bitmagic: binary search through a uint32, offset down by 1 to not round powers of 2 up.
	// Then adding 1 to it to not get the index of the first bit, but the length of the bits (depth of tree)
	// Zero is a special case, it has a 0 depth.
	// Example:
	//  (in out): (0 0), (1 1), (2 1), (3 2), (4 2), (5 3), (6 3), (7 3), (8 3), (9 4)
	if v == 0 {
		return 0
	}
	v--
	if v&mask5 != 0 {
		v >>= bit5
		out |= bit5
	}
	if v&mask4 != 0 {
		v >>= bit4
		out |= bit4
	}
	if v&mask3 != 0 {
		v >>= bit3
		out |= bit3
	}
	if v&mask2 != 0 {
		v >>= bit2
		out |= bit2
	}
	if v&mask1 != 0 {
		v >>= bit1
		out |= bit1
	}
	if v&mask0 != 0 {
		out |= bit0
	}
	out++
	return
}

// Merkleize with log(N) space allocation
func Merkleize(hasher Hasher, count uint64, limit uint64, leaf func(i uint64) []byte) (out [32]byte) {
	if count > limit {
		panic("merkleizing list that is too large, over limit")
	}
	if limit == 0 {
		return
	}
	if limit == 1 {
		if count == 1 {
			copy(out[:], leaf(0))
		}
		return
	}
	depth := GetDepth(count)
	limitDepth := GetDepth(limit)
	tmp := make([][32]byte, limitDepth+1)

	j := uint8(0)
	hArr := [32]byte{}
	h := hArr[:]

	merge := func(i uint64) {
		// merge back up from bottom to top, as far as we can
		for j = 0; ; j++ {
			// stop merging when we are in the left side of the next combi
			if i&(uint64(1)<<j) == 0 {
				// if we are at the count, we want to merge in zero-hashes for padding
				if i == count && j < depth {
					v := hasher.Combi(hArr, trieutil.ZeroHashes[j])
					copy(h, v[:])
				} else {
					break
				}
			} else {
				// keep merging up if we are the right side
				v := hasher.Combi(tmp[j], hArr)
				copy(h, v[:])
			}
		}
		// store the merge result (may be no merge, i.e. bottom leaf node)
		copy(tmp[j][:], h)
	}

	// merge in leaf by leaf.
	for i := uint64(0); i < count; i++ {
		copy(h[:], leaf(i))
		merge(i)
	}

	// complement with 0 if empty, or if not the right power of 2
	if (uint64(1) << depth) != count {
		copy(h[:], trieutil.ZeroHashes[0][:])
		merge(count)
	}

	// the next power of two may be smaller than the ultimate virtual size,
	// complement with zero-hashes at each depth.
	for j := depth; j < limitDepth; j++ {
		tmp[j+1] = hasher.Combi(tmp[j], trieutil.ZeroHashes[j])
	}

	return tmp[limitDepth]
}

// ConstructProof builds a merkle-branch of the given depth, at the given index (at that depth),
// for a list of leafs of a balanced binary tree.
func ConstructProof(hasher Hasher, count uint64, limit uint64, leaf func(i uint64) []byte, index uint64) (branch [][32]byte) {
	if count > limit {
		panic("merkleizing list that is too large, over limit")
	}
	if index >= limit {
		panic("index out of range, over limit")
	}
	if limit <= 1 {
		return
	}
	depth := GetDepth(count)
	limitDepth := GetDepth(limit)
	branch = append(branch, trieutil.ZeroHashes[:limitDepth]...)

	tmp := make([][32]byte, limitDepth+1)

	j := uint8(0)
	hArr := [32]byte{}
	h := hArr[:]

	merge := func(i uint64) {
		// merge back up from bottom to top, as far as we can
		for j = 0; ; j++ {
			// if i is a sibling of index at the given depth,
			// and i is the last index of the subtree to that depth,
			// then put h into the branch
			if (i>>j)^1 == (index>>j) && (((1<<j)-1)&i) == ((1<<j)-1) {
				// insert sibling into the proof
				branch[j] = hArr
			}
			// stop merging when we are in the left side of the next combi
			if i&(uint64(1)<<j) == 0 {
				// if we are at the count, we want to merge in zero-hashes for padding
				if i == count && j < depth {
					v := hasher.Combi(hArr, trieutil.ZeroHashes[j])
					copy(h, v[:])
				} else {
					break
				}
			} else {
				// keep merging up if we are the right side
				v := hasher.Combi(tmp[j], hArr)
				copy(h, v[:])
			}
		}
		// store the merge result (may be no merge, i.e. bottom leaf node)
		copy(tmp[j][:], h)
	}

	// merge in leaf by leaf.
	for i := uint64(0); i < count; i++ {
		copy(h[:], leaf(i))
		merge(i)
	}

	// complement with 0 if empty, or if not the right power of 2
	if (uint64(1) << depth) != count {
		copy(h[:], trieutil.ZeroHashes[0][:])
		merge(count)
	}

	return
}
