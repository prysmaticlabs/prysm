// Package slasher implements slashing validation
// and proof creation.
package slasher

import (
	"github.com/prysmaticlabs/go-bitfield"
)

var epochProposalBitlist map[uint64]bitfield.Bitlist

func init() {
	epochProposalBitlist = make(map[uint64]bitfield.Bitlist, 54000)
}

// CheckNewProposal checks weather a new proposal is allowed or
// creating a slashable event.
// returns true if it is the first time this
// validatorID propose a block in this epoch or not.
func CheckNewProposal(epoch uint64, validatorID uint64) bool {
	proposalExists := epochProposalBitlist[epoch].BitAt(validatorID)
	if !proposalExists {
		epochProposalBitlist[epoch].SetBitAt(validatorID, true)
		return true
	}
	return false
}
