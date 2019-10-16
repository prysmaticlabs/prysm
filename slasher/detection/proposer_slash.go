// Package slasher implements slashing validation
// and proof creation.
package slasher

import (
	"errors"
	"github.com/prysmaticlabs/prysm/shared/params"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

var epochProposalBitlist map[uint64]bitfield.Bitlist
var currentEpoch, weakSubjectivityPeriod uint64
var epochs []uint64

func init() {
	epochProposalBitlist = make(map[uint64]bitfield.Bitlist)
	weakSubjectivityPeriod = params.BeaconConfig().WeakSubjectivityPeriod
}

// CheckNewProposal checks weather a new proposal is allowed or
// creating a slashable event.
// returns true if it is the first time this
// validatorID propose a block in this epoch or not.
func CheckNewProposal(currentEpoch uint64, epoch uint64, validatorID uint64) (bool, error) {
	if currentEpoch > weakSubjectivityPeriod && epoch < currentEpoch-weakSubjectivityPeriod {
		return false, errors.New("epoch is obsolete = before weak subjectivity period")
	}

	if _, ok := epochProposalBitlist[epoch]; !ok {
		epochProposalBitlist[epoch] = bitfield.NewBitlist(300000)
		epochs = sliceutil.InsertSort(epochs, epoch)
		var itemsToTruncate []uint64
		if currentEpoch > weakSubjectivityPeriod {
			itemsToTruncate = sliceutil.TruncateItems(epochs, currentEpoch-weakSubjectivityPeriod)
			for _, key := range itemsToTruncate {
				delete(epochProposalBitlist, key)
			}
		}
	}
	proposalExists := epochProposalBitlist[epoch].BitAt(validatorID)
	if !proposalExists {
		epochProposalBitlist[epoch].SetBitAt(validatorID, true)
		return true, nil
	}
	return false, nil
}
