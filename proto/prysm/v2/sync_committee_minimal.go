// +build minimal

package v2

import (
	"github.com/prysmaticlabs/go-bitfield"
)

func NewSyncCommitteeAggregationBits() bitfield.Bitvector8 {
	return bitfield.NewBitvector8()
}
