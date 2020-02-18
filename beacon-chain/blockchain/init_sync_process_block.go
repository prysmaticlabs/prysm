package blockchain

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const maxCacheSize = 70
const initialSyncCacheSize = 45
const minimumCacheSize = initialSyncCacheSize / 3

func (s *Service) persistCachedStates(ctx context.Context, numOfStates int) error {
	oldStates := make([]*stateTrie.BeaconState, 0, numOfStates)

	// Add slots to the map and add epoch boundary states to the slice.
	for _, rt := range s.boundaryRoots[:numOfStates-minimumCacheSize] {
		oldStates = append(oldStates, s.initSyncState[rt])
	}

	err := s.beaconDB.SaveStates(ctx, oldStates, s.boundaryRoots[:numOfStates-minimumCacheSize])
	if err != nil {
		return err
	}
	for _, rt := range s.boundaryRoots[:numOfStates-minimumCacheSize] {
		delete(s.initSyncState, rt)
	}
	s.boundaryRoots = s.boundaryRoots[numOfStates-minimumCacheSize:]
	return nil
}

// filter out boundary candidates from our currently processed batch of states.
func (s *Service) filterBoundaryCandidates(ctx context.Context, root [32]byte, postState *stateTrie.BeaconState) {
	// Only trigger on epoch start.
	if !helpers.IsEpochStart(postState.Slot()) {
		return
	}

	stateSlice := make([][32]byte, 0, len(s.initSyncState))
	// Add epoch boundary roots to slice.
	for rt := range s.initSyncState {
		stateSlice = append(stateSlice, rt)
	}

	sort.Slice(stateSlice, func(i int, j int) bool {
		return s.initSyncState[stateSlice[i]].Slot() < s.initSyncState[stateSlice[j]].Slot()
	})
	epochLength := params.BeaconConfig().SlotsPerEpoch

	if len(s.boundaryRoots) > 0 {
		// Retrieve previous boundary root.
		previousBoundaryRoot := s.boundaryRoots[len(s.boundaryRoots)-1]
		previousState, ok := s.initSyncState[previousBoundaryRoot]
		if !ok {
			// Remove the non-existent root and exit filtering.
			s.boundaryRoots = s.boundaryRoots[:len(s.boundaryRoots)-1]
			return
		}
		previousSlot := previousState.Slot()

		// Round up slot number to account for skipped slots.
		previousSlot = helpers.RoundUpToNearestEpoch(previousSlot)
		if postState.Slot()-previousSlot >= epochLength {
			targetSlot := postState.Slot()
			tempRoots := s.loopThroughCandidates(stateSlice, previousBoundaryRoot, previousSlot, targetSlot)
			s.boundaryRoots = append(s.boundaryRoots, tempRoots...)
		}
	}
	s.boundaryRoots = append(s.boundaryRoots, root)
	s.pruneOldStates()
	s.pruneNonBoundaryStates()
}

// loop-through the provided candidate roots to filter out which would be appropriate boundary roots.
func (s *Service) loopThroughCandidates(stateSlice [][32]byte, previousBoundaryRoot [32]byte,
	previousSlot uint64, targetSlot uint64) [][32]byte {
	tempRoots := [][32]byte{}
	epochLength := params.BeaconConfig().SlotsPerEpoch

	// Loop through current states to filter for valid boundary states.
	for i := len(stateSlice) - 1; stateSlice[i] != previousBoundaryRoot && i >= 0; i-- {
		currentSlot := s.initSyncState[stateSlice[i]].Slot()
		// Skip if the current slot is larger than the previous epoch
		// boundary.
		if currentSlot > targetSlot-epochLength {
			continue
		}
		tempRoots = append(tempRoots, stateSlice[i])

		// Switch target slot if the current slot is greater than
		// 1 epoch boundary from the previously saved boundary slot.
		if currentSlot > previousSlot+epochLength {
			currentSlot = helpers.RoundUpToNearestEpoch(currentSlot)
			targetSlot = currentSlot
			continue
		}
		break
	}
	// Reverse to append the roots in ascending order corresponding
	// to the respective slots.
	tempRoots = bytesutil.ReverseBytes32Slice(tempRoots)
	return tempRoots
}

// prune for states past the current finalized checkpoint.
func (s *Service) pruneOldStates() {
	prunedBoundaryRoots := [][32]byte{}
	for _, rt := range s.boundaryRoots {
		st, ok := s.initSyncState[rt]
		// Skip non-existent roots.
		if !ok {
			continue
		}
		if st.Slot() < helpers.StartSlot(s.FinalizedCheckpt().Epoch) {
			delete(s.initSyncState, rt)
			continue
		}
		prunedBoundaryRoots = append(prunedBoundaryRoots, rt)
	}
	s.boundaryRoots = prunedBoundaryRoots
}

// prune cache for non-boundary states.
func (s *Service) pruneNonBoundaryStates() {
	boundaryMap := make(map[[32]byte]bool)
	for i := range s.boundaryRoots {
		boundaryMap[s.boundaryRoots[i]] = true
	}
	for rt := range s.initSyncState {
		if !boundaryMap[rt] {
			delete(s.initSyncState, rt)
		}
	}
}

func (s *Service) pruneOldNonFinalizedStates() {
	stateSlice := make([][32]byte, 0, len(s.initSyncState))
	// Add epoch boundary roots to slice.
	for rt := range s.initSyncState {
		stateSlice = append(stateSlice, rt)
	}

	// Sort by slots.
	sort.Slice(stateSlice, func(i int, j int) bool {
		return s.initSyncState[stateSlice[i]].Slot() < s.initSyncState[stateSlice[j]].Slot()
	})

	boundaryMap := make(map[[32]byte]bool)
	for i := range s.boundaryRoots {
		boundaryMap[s.boundaryRoots[i]] = true
	}
	for _, rt := range stateSlice[:initialSyncCacheSize] {
		if boundaryMap[rt] {
			continue
		}
		delete(s.initSyncState, rt)
	}
}

func (s *Service) generateState(ctx context.Context, startRoot [32]byte, endRoot [32]byte) (*stateTrie.BeaconState, error) {
	preState, err := s.beaconDB.State(ctx, startRoot)
	if err != nil {
		return nil, err
	}
	if preState == nil {
		return nil, errors.New("finalized state does not exist in db")
	}
	endBlock, err := s.beaconDB.Block(ctx, endRoot)
	if err != nil {
		return nil, err
	}
	if endBlock == nil {
		return nil, errors.New("provided block root does not have block saved in the db")
	}
	log.Warnf("Generating missing state of slot %d and root %#x", endBlock.Block.Slot, endRoot)

	blocks, err := s.stateGen.LoadBlocks(ctx, preState.Slot()+1, endBlock.Block.Slot, endRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not load the required blocks")
	}
	postState, err := s.stateGen.ReplayBlocks(ctx, preState, blocks, endBlock.Block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not replay the blocks to generate the resultant state")
	}
	return postState, nil
}
