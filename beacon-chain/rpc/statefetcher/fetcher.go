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

// Fetcher is responsible for retrieving the BeaconState.
type Fetcher interface {
	State(ctx context.Context, stateId []byte) (iface.BeaconState, error)
}

// StateProvider is a real implementation of Fetcher.
type StateProvider struct {
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
func (p *StateProvider) State(ctx context.Context, stateId []byte) (iface.BeaconState, error) {
	var (
		s   iface.BeaconState
		err error
	)

	stateIdString := strings.ToLower(string(stateId))
	switch stateIdString {
	case "head":
		s, err = p.ChainInfoFetcher.HeadState(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get head state")
		}
	case "genesis":
		s, err = p.BeaconDB.GenesisState(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get genesis state")
		}
	case "finalized":
		checkpoint := p.ChainInfoFetcher.FinalizedCheckpt()
		s, err = p.StateGenService.StateByRoot(ctx, bytesutil.ToBytes32(checkpoint.Root))
		if err != nil {
			return nil, errors.Wrap(err, "could not get finalized state")
		}
	case "justified":
		checkpoint := p.ChainInfoFetcher.CurrentJustifiedCheckpt()
		s, err = p.StateGenService.StateByRoot(ctx, bytesutil.ToBytes32(checkpoint.Root))
		if err != nil {
			return nil, errors.Wrap(err, "could not get justified state")
		}
	default:
		ok, matchErr := bytesutil.IsBytes32Hex(stateId)
		if matchErr != nil {
			return nil, errors.Wrap(err, "could not parse ID")
		}
		if ok {
			s, err = p.stateByHex(ctx, stateId)
		} else {
			slotNumber, parseErr := strconv.ParseUint(stateIdString, 10, 64)
			if parseErr != nil {
				// ID format does not match any valid options.
				return nil, errors.New("invalid state ID: " + stateIdString)
			}
			s, err = p.stateBySlot(ctx, types.Slot(slotNumber))
		}
	}

	return s, err
}

func (p *StateProvider) stateByHex(ctx context.Context, stateId []byte) (iface.BeaconState, error) {
	headState, err := p.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state")
	}
	for i, root := range headState.StateRoots() {
		if bytes.Equal(root, stateId) {
			blockRoot := headState.BlockRoots()[i]
			return p.StateGenService.StateByRoot(ctx, bytesutil.ToBytes32(blockRoot))
		}
	}
	return nil, fmt.Errorf("state not found in the last %d state roots in head state", len(headState.StateRoots()))
}

func (p *StateProvider) stateBySlot(ctx context.Context, slot types.Slot) (iface.BeaconState, error) {
	currentSlot := p.GenesisTimeFetcher.CurrentSlot()
	if slot > currentSlot {
		return nil, errors.New("slot cannot be in the future")
	}
	state, err := p.StateGenService.StateBySlot(ctx, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state")
	}
	return state, nil
}
