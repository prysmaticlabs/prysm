package borrower

import "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice"

func NewForkchoiceBorrower(fc forkchoice.ForkChoicer) *ForkchoiceBorrower {
	return &ForkchoiceBorrower{fc: fc}
}

type ForkchoiceBorrower struct {
	fc forkchoice.ForkChoicer
}

type BorrowedForkChoicer struct {
	forkchoice.ForkChoicer
	unlock func()
}

// Return ends the borrow lifecycle, calling the (Unlock/RUlock) method and nilling the
// pointer to the underlying forkchoice object so that it can no longer be used by the caller.
func (b *BorrowedForkChoicer) Return() {
	if b.unlock != nil {
		b.unlock()
	}
	b.ForkChoicer = nil
}

// Borrow is used to borrow the Forkchoicer. When the borrower is done with it, they must
// call the function, passing in the returned Forkchoicer, to invalidate the pointer.
func (fb *ForkchoiceBorrower) Borrow() (*BorrowedForkChoicer, func()) {
	fb.fc.Lock()
	b := &BorrowedForkChoicer{ForkChoicer: fb.fc, unlock: fb.fc.Unlock}
	return b, b.Return
}

// BorrowRO is an implementation of Borrow that uses read locks rather than write locks.
// Borrow is used to borrow the Forkchoicer. When the borrower is done with it, they must
// call the function, passing in the returned Forkchoicer, to invalidate the pointer.
func (fb *ForkchoiceBorrower) BorrowRO() (*BorrowedForkChoicer, func()) {
	fb.fc.RLock()
	b := &BorrowedForkChoicer{ForkChoicer: fb.fc, unlock: fb.fc.RUnlock}
	return b, b.Return
}
