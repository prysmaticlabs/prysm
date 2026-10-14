package stateutil

import (
	"github.com/OffchainLabs/go-bitfield"
	"github.com/OffchainLabs/methodical-ssz/ssz"
)

type progressiveBitlist []byte

// HashTreeRoot hashes the bitlist as a progressive list.
func (p progressiveBitlist) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(p)
}

// HashTreeRootWith hashes the bitlist as a progressive list with the given hasher.
func (p progressiveBitlist) HashTreeRootWith(hh *ssz.Hasher) error {
	hh.PutProgressiveBitlist(p)
	return nil
}

// ProgressiveBitlistRoot is the root of a progressive bitlist. The generated state hashes these fields with PutProgressiveBitlist, so the field root must match it.
func ProgressiveBitlistRoot(bits bitfield.Bitlist) ([32]byte, error) {
	return progressiveBitlist(bits).HashTreeRoot()
}
