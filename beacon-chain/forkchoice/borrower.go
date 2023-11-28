package forkchoice

func NewBorrower(fc ForkChoicer) *Borrower {
	return &Borrower{fc: fc}
}

type Borrower struct {
	fc ForkChoicer
}

type Borrowed struct {
	ForkChoicer
	unlock func()
}

// Return ends the borrow lifecycle, calling the (Unlock/RUlock) method and nilling the
// pointer to the underlying forkchoice object so that it can no longer be used by the caller.
func (b *Borrowed) Return() {
	if b.unlock != nil {
		b.unlock()
	}
	b.ForkChoicer = nil
}

// Borrow is used to borrow the Forkchoicer. When the borrower is done with it, they must
// call the function, passing in the returned Forkchoicer, to invalidate the pointer.
func (fb *Borrower) Borrow() (*Borrowed, func()) {
	fb.fc.Lock()
	b := &Borrowed{ForkChoicer: fb.fc, unlock: fb.fc.Unlock}
	return b, b.Return
}

// BorrowRO is an implementation of Borrow that uses read locks rather than write locks.
// Borrow is used to borrow the Forkchoicer. When the borrower is done with it, they must
// call the function, passing in the returned Forkchoicer, to invalidate the pointer.
func (fb *Borrower) RBorrow() (*Borrowed, func()) {
	fb.fc.RLock()
	b := &Borrowed{ForkChoicer: fb.fc, unlock: fb.fc.RUnlock}
	return b, b.Return
}
