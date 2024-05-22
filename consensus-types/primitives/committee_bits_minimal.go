//go:build minimal

package primitives

func NewAttestationCommitteeBits() bitfield.Bitvector4 {
	return bitfield.NewBitvector4()
}
