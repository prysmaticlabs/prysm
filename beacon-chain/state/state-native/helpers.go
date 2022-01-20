package state_native

import customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/custom-types"

// BlockRootsToSlice converts a customtypes.BlockRoots object into a 2D byte slice.
func BlockRootsToSlice(blockRoots *customtypes.BlockRoots) [][]byte {
	if blockRoots == nil {
		return nil
	}
	var bRoots [][]byte
	bRoots = make([][]byte, len(blockRoots))
	for i, r := range blockRoots {
		tmp := r
		bRoots[i] = tmp[:]
	}
	return bRoots
}

// StateRootsToSlice converts a customtypes.StateRoots object into a 2D byte slice.
func StateRootsToSlice(stateRoots *customtypes.StateRoots) [][]byte {
	if stateRoots == nil {
		return nil
	}
	var sRoots [][]byte
	sRoots = make([][]byte, len(stateRoots))
	for i, r := range stateRoots {
		tmp := r
		sRoots[i] = tmp[:]
	}
	return sRoots
}

// HistoricalRootsToSlice converts a customtypes.HistoricalRoots object into a 2D byte slice.
func HistoricalRootsToSlice(historicalRoots customtypes.HistoricalRoots) [][]byte {
	if historicalRoots == nil {
		return nil
	}
	var hRoots [][]byte
	hRoots = make([][]byte, len(historicalRoots))
	for i, r := range historicalRoots {
		tmp := r
		hRoots[i] = tmp[:]
	}
	return hRoots
}

// RandaoMixesToSlice converts a customtypes.RandaoMixes object into a 2D byte slice.
func RandaoMixesToSlice(randaoMixes *customtypes.RandaoMixes) [][]byte {
	if randaoMixes == nil {
		return nil
	}
	var mixes [][]byte
	mixes = make([][]byte, len(randaoMixes))
	for i, r := range randaoMixes {
		tmp := r
		mixes[i] = tmp[:]
	}
	return mixes
}
