package bitfield

var _ = Bitfield(Bitvector{})

type Bitvector []byte

func (b Bitvector) BitAt(idx uint64) bool {
	return false
}

func (b Bitvector) SetBitAt(idx uint64, val bool) {

}

func (b Bitvector) Len() uint64 {
	return 0
}
