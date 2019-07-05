package bitfield

type Bitfield interface {
	BitAt(idx uint64) bool
	SetBitAt(idx uint64, val bool)
	Len() uint64
}
