package p2p

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

// filterNodes wraps an iterator such that Next only returns nodes for which
// the 'check' function returns true. This custom implementation also
// checks for context deadlines so that t
func filterNodes(ctx context.Context, it enode.Iterator, check func(*enode.Node) bool) enode.Iterator {
	return &filterIter{ctx, it, check}
}

type filterIter struct {
	context.Context
	enode.Iterator
	check func(*enode.Node) bool
}

func (f *filterIter) Next() bool {
	for f.Iterator.Next() {
		if f.Context.Err() != nil {
			return false
		}
		if f.check(f.Node()) {
			return true
		}
	}
	return false
}
