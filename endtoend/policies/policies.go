package policies

// AfterNthEpoch runs for every epoch after the provided epoch.
func AfterNthEpoch(afterEpoch uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return currentEpoch > afterEpoch
	}
}

// AllEpochs runs for all epochs.
func AllEpochs(_ uint64) bool {
	return true
}

// OnEpoch runs only for the provided epoch.
func OnEpoch(epoch uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return currentEpoch == epoch
	}
}

// BetweenEpochs runs for every epoch that is between the provided epochs.
func BetweenEpochs(fromEpoch, toEpoch uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return fromEpoch < currentEpoch && currentEpoch < toEpoch
	}
}
