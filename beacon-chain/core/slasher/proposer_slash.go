// Package slasher implements slashing validation
// and proof creation.
package slasher

import (
	"sort"

	"github.com/prysmaticlabs/go-bitfield"
)

var epochProposalBitlist map[uint64]bitfield.Bitlist
var currentEpoch, weakSubjectivityPeriod uint64
var epochs []uint64

func init() {
	epochProposalBitlist = make(map[uint64]bitfield.Bitlist)
	weakSubjectivityPeriod = uint64(54000)

}

// CheckNewProposal checks weather a new proposal is allowed or
// creating a slashable event.
// returns true if it is the first time this
// validatorID propose a block in this epoch or not.
func CheckNewProposal(currentEpoch uint64, epoch uint64, validatorID uint64) bool {

	_, ok := epochProposalBitlist[epoch]
	if !ok {

		epochProposalBitlist[epoch] = bitfield.NewBitlist(300000)
	}
	proposalExists := epochProposalBitlist[epoch].BitAt(validatorID)
	if !proposalExists {
		epochProposalBitlist[epoch].SetBitAt(validatorID, true)
		return true
	}
	return false
}

func insertSort(data []uint64, u uint64) []uint64 {
	index := sort.Search(len(data), func(i int) bool { return data[i] > u })
	data = append(data, uint64(0))
	copy(data[index+1:], data[index:])
	data[index] = u
	return data
}
