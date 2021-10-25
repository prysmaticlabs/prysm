package light

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/config/params"
)

func isBetterUpdate(prevUpdate *ClientUpdate, newUpdate *ClientUpdate) bool {
	prevIsFinalized := isFinalizedUpdate(prevUpdate)
	newIsFinalized := isFinalizedUpdate(newUpdate)
	// newUpdate becomes finalized, it's better.
	if newIsFinalized && !prevIsFinalized {
		return true
	}
	// newUpdate is no longer finalized, it's worse.
	if !newIsFinalized && prevIsFinalized {
		return false
	}
	return hasMoreBits(newUpdate, prevUpdate)
}

func isLatestBestFinalizedUpdate(prevUpdate *ClientUpdate, newUpdate *ClientUpdate) bool {
	if newUpdate.FinalityHeader.Slot > prevUpdate.FinalityHeader.Slot {
		return true
	}
	if newUpdate.FinalityHeader.Slot < prevUpdate.FinalityHeader.Slot {
		return false
	}
	return hasMoreBits(newUpdate, prevUpdate)
}

func isLatestBestNonFinalizedUpdate(prevUpdate *ClientUpdate, newUpdate *ClientUpdate) bool {
	if newUpdate.Header.Slot > prevUpdate.Header.Slot {
		return true
	}
	if newUpdate.Header.Slot < prevUpdate.Header.Slot {
		return false
	}
	return hasMoreBits(newUpdate, prevUpdate)
}

func isFinalizedUpdate(update *ClientUpdate) bool {
	return !bytes.Equal(params.BeaconConfig().ZeroHash[:], update.FinalityHeader.StateRoot)
}

func hasMoreBits(a *ClientUpdate, b *ClientUpdate) bool {
	return a.SyncCommitteeBits.Count() > b.SyncCommitteeBits.Count()
}
