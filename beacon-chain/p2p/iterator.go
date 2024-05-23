package p2p

import (
	"context"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

const backOffCounter = 50

// filterNodes wraps an iterator such that Next only returns nodes for which
// the 'check' function returns true. This custom implementation also
// checks for context deadlines so that in the event the parent context has
// expired, we do exit from the search rather than  perform more network
// lookups for additional peers.
func filterNodes(ctx context.Context, it enode.Iterator, check func(*enode.Node) bool) enode.Iterator {
	return &filterIter{ctx, it, check}
}

type filterIter struct {
	context.Context
	enode.Iterator
	check func(*enode.Node) bool
}

// Next looks up for the next valid node according to our
// filter criteria.
func (f *filterIter) Next() bool {
	lookupCounter := 0
	start := time.Now()
	for f.Iterator.Next() {
		if time.Since(start) > 3*time.Second {
			log.Infof("query takes %s", time.Since(start).String())
		}
		if time.Since(start) > 5*time.Second {
			log.Infof("very long query %s", time.Since(start).String())
			runtime.Gosched()
			time.Sleep(30 * time.Second)
		}
		// Do not excessively perform lookups if we constantly receive non-viable peers.
		if lookupCounter > backOffCounter {
			lookupCounter = 0
			runtime.Gosched()
			time.Sleep(30 * time.Second)
		}
		if f.Context.Err() != nil {
			return false
		}
		if f.check(f.Node()) {
			return true
		}
		lookupCounter++
		start = time.Now()
	}
	return false
}
