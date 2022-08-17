package policies

import types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"

// AfterNthEpoch runs for every epoch after the provided epoch.
func AfterNthEpoch(afterEpoch types.Epoch) func(epoch types.Epoch) bool {
	return func(currentEpoch types.Epoch) bool {
		return currentEpoch > afterEpoch
	}
}

// AllEpochs runs for all epochs.
func AllEpochs(_ types.Epoch) bool {
	return true
}

// OnEpoch runs only for the provided epoch.
func OnEpoch(epoch types.Epoch) func(types.Epoch) bool {
	return func(currentEpoch types.Epoch) bool {
		return currentEpoch == epoch
	}
}

// BetweenEpochs runs for every epoch that is between the provided epochs.
func BetweenEpochs(fromEpoch, toEpoch types.Epoch) func(types.Epoch) bool {
	return func(currentEpoch types.Epoch) bool {
		return fromEpoch < currentEpoch && currentEpoch < toEpoch
	}
}
