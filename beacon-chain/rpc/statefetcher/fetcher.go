package statefetcher

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// StateFetcher is responsible for retrieving the BeaconState.
type StateFetcher struct {
	BeaconDB           db.ReadOnlyDatabase
	ChainInfoFetcher   blockchain.ChainInfoFetcher
	GenesisTimeFetcher blockchain.TimeFetcher
	StateGenService    stategen.StateManager
}

// State returns the BeaconState for a given identifier. The identifier can be one of:
//  - "head" (canonical head in node's view)
//  - "genesis"
//  - "finalized"
//  - "justified"
//  - <slot>
//  - <hex encoded stateRoot with 0x prefix>
func (f *StateFetcher) State(ctx context.Context, stateId []byte) (iface.BeaconState, error) {
	var (
		s   iface.BeaconState
		err error
	)

	stateIdString := strings.ToLower(string(stateId))
	switch stateIdString {
	case "head":
		s, err = f.ChainInfoFetcher.HeadState(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get head state")
		}
	case "genesis":
		s, err = f.BeaconDB.GenesisState(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get genesis state")
		}
	case "finalized":
		checkpoint := f.ChainInfoFetcher.FinalizedCheckpt()
		s, err = f.StateGenService.StateByRoot(ctx, bytesutil.ToBytes32(checkpoint.Root))
		if err != nil {
			return nil, errors.Wrap(err, "could not get finalized state")
		}
	case "justified":
		checkpoint := f.ChainInfoFetcher.CurrentJustifiedCheckpt()
		s, err = f.StateGenService.StateByRoot(ctx, bytesutil.ToBytes32(checkpoint.Root))
		if err != nil {
			return nil, errors.Wrap(err, "could not get justified state")
		}
	default:
		ok, matchErr := bytesutil.IsBytes32Hex(stateId)
		if matchErr != nil {
			return nil, errors.Wrap(err, "could not parse ID")
		}
		if ok {
			s, err = f.stateByHex(ctx, stateId)
		} else {
			slotNumber, parseErr := strconv.ParseUint(stateIdString, 10, 64)
			if parseErr != nil {
				// ID format does not match any valid options.
				return nil, errors.New("invalid state ID: " + stateIdString)
			}
			s, err = f.stateBySlot(ctx, types.Slot(slotNumber))
		}
	}

	return s, err
}

func (f *StateFetcher) stateByHex(ctx context.Context, stateId []byte) (iface.BeaconState, error) {
	headState, err := f.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state")
	}
	for i, root := range headState.StateRoots() {
		if bytes.Equal(root, stateId) {
			blockRoot := headState.BlockRoots()[i]
			return f.StateGenService.StateByRoot(ctx, bytesutil.ToBytes32(blockRoot))
		}
	}
	return nil, fmt.Errorf("state not found in the last %d state roots in head state", len(headState.StateRoots()))
}

func (f *StateFetcher) stateBySlot(ctx context.Context, slot types.Slot) (iface.BeaconState, error) {
	currentSlot := f.GenesisTimeFetcher.CurrentSlot()
	if slot > currentSlot {
		return nil, errors.New("slot cannot be in the future")
	}
	state, err := f.StateGenService.StateBySlot(ctx, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state")
	}
	return state, nil
}
