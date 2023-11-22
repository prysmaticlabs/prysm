package borrower_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/borrower"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestBorrowNilness(t *testing.T) {
	realFC := doublylinkedtree.New()
	fb := borrower.NewForkchoiceBorrower(realFC)
	fc, r := fb.Borrow()
	// just calling a method to make sure this doesn't panic
	require.Equal(t, 0, fc.NodeCount())
	r()
	require.Equal(t, nil, fc.ForkChoicer)
}
