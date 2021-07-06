// +build !minimal

package v2

import (
	"github.com/prysmaticlabs/go-bitfield"
)

func NewSyncCommitteeAggregationBits() bitfield.Bitvector128 {
	return bitfield.NewBitvector128()
}
