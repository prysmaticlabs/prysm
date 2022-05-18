//go:build minimal
// +build minimal

package eth

import (
	"github.com/prysmaticlabs/go-bitfield"
)

func NewSyncCommitteeAggregationBits() bitfield.Bitvector8 {
	return bitfield.NewBitvector8()
}

func ConvertSyncContributionBitVector(b []byte) bitfield.Bitvector8 {
	return b
}
