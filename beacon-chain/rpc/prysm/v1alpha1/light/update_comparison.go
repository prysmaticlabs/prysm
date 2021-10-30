package light

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

func isBetterUpdate(prevUpdate *ethpb.LightClientUpdate, newUpdate *ethpb.LightClientUpdate) bool {
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

func isLatestBestFinalizedUpdate(prevUpdate *ethpb.LightClientUpdate, newUpdate *ethpb.LightClientUpdate) bool {
	if newUpdate.FinalityHeader.Slot > prevUpdate.FinalityHeader.Slot {
		return true
	}
	if newUpdate.FinalityHeader.Slot < prevUpdate.FinalityHeader.Slot {
		return false
	}
	return hasMoreBits(newUpdate, prevUpdate)
}

func isLatestBestNonFinalizedUpdate(prevUpdate *ethpb.LightClientUpdate, newUpdate *ethpb.LightClientUpdate) bool {
	if newUpdate.Header.Slot > prevUpdate.Header.Slot {
		return true
	}
	if newUpdate.Header.Slot < prevUpdate.Header.Slot {
		return false
	}
	return hasMoreBits(newUpdate, prevUpdate)
}

func isFinalizedUpdate(update *ethpb.LightClientUpdate) bool {
	return !bytes.Equal(params.BeaconConfig().ZeroHash[:], update.FinalityHeader.StateRoot)
}

func hasMoreBits(a *ethpb.LightClientUpdate, b *ethpb.LightClientUpdate) bool {
	return a.SyncCommitteeBits.Count() > b.SyncCommitteeBits.Count()
}
