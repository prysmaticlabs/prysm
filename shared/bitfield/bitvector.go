package bitfield

var _ = Bitfield(Bitvector4{})

type Bitvector4 []byte

func (b Bitvector4) BitAt(idx uint64) bool {
	// Out of bounds, must be false.
	upperBounds := b.Len()
	if idx > upperBounds || upperBounds == 0 {
		return false
	}

	i := uint8(1 << idx)
	return b[0]&i == i

}

func (b Bitvector4) SetBitAt(idx uint64, val bool) {
	// Out of bounds, do nothing.
	upperBounds := b.Len()
	if idx > upperBounds || upperBounds == 0 {
		return
	}

	bit := uint8(1 << idx)
	if val {
		b[0] |= bit
	} else {
		b[0] &^= bit
	}
}

func (b Bitvector4) Len() uint64 {
	return 4
}
