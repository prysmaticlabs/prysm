package statefetcher

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"go.opencensus.io/trace"
)

// StateIdParseError represents an error scenario where a state ID could not be parsed.
type StateIdParseError struct {
	message string
}

// NewStateIdParseError creates a new error instance.
func NewStateIdParseError(reason error) StateIdParseError {
	return StateIdParseError{
		message: errors.Wrapf(reason, "could not parse state ID").Error(),
	}
}

// Error returns the underlying error message.
func (e *StateIdParseError) Error() string {
	return e.message
}

// StateNotFoundError represents an error scenario where a state could not be found.
type StateNotFoundError struct {
	message string
}

// NewStateNotFoundError creates a new error instance.
func NewStateNotFoundError(stateRootsSize int) StateNotFoundError {
	return StateNotFoundError{
		message: fmt.Sprintf("state not found in the last %d state roots", stateRootsSize),
	}
}

// Error returns the underlying error message.
func (e *StateNotFoundError) Error() string {
	return e.message
}

// StateRootNotFoundError represents an error scenario where a state root could not be found.
type StateRootNotFoundError struct {
	message string
}

// NewStateRootNotFoundError creates a new error instance.
func NewStateRootNotFoundError(stateRootsSize int) StateNotFoundError {
	return StateNotFoundError{
		message: fmt.Sprintf("state root not found in the last %d state roots", stateRootsSize),
	}
}

// Error returns the underlying error message.
func (e *StateRootNotFoundError) Error() string {
	return e.message
}

// Fetcher is responsible for retrieving info related with the beacon chain.
type Fetcher interface {
	State(ctx context.Context, stateId []byte) (state.BeaconState, error)
	StateRoot(ctx context.Context, stateId []byte) ([]byte, error)
	StateBySlot(ctx context.Context, slot types.Slot) (state.BeaconState, error)
}

// StateProvider is a real implementation of Fetcher.
type StateProvider struct {
	BeaconDB           db.ReadOnlyDatabase
	ChainInfoFetcher   blockchain.ChainInfoFetcher
	GenesisTimeFetcher blockchain.TimeFetcher
	StateGenService    stategen.StateManager
	ReplayerBuilder    stategen.ReplayerBuilder
}

// State returns the BeaconState for a given identifier. The identifier can be one of:
//  - "head" (canonical head in node's view)
//  - "genesis"
//  - "finalized"
//  - "justified"
//  - <slot>
//  - <hex encoded state root with '0x' prefix>
func (p *StateProvider) State(ctx context.Context, stateId []byte) (state.BeaconState, error) {
	var (
		s   state.BeaconState
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
		s, err = p.StateBySlot(ctx, params.BeaconConfig().GenesisSlot)
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
		if len(stateId) == 32 {
			s, err = p.stateByRoot(ctx, stateId)
		} else {
			slotNumber, parseErr := strconv.ParseUint(stateIdString, 10, 64)
			if parseErr != nil {
				// ID format does not match any valid options.
				e := NewStateIdParseError(parseErr)
				return nil, &e
			}
			s, err = p.StateBySlot(ctx, types.Slot(slotNumber))
		}
	}

	return s, err
}

// StateRoot returns a beacon state root for a given identifier. The identifier can be one of:
//  - "head" (canonical head in node's view)
//  - "genesis"
//  - "finalized"
//  - "justified"
//  - <slot>
//  - <hex encoded state root with '0x' prefix>
func (p *StateProvider) StateRoot(ctx context.Context, stateId []byte) (root []byte, err error) {
	stateIdString := strings.ToLower(string(stateId))
	switch stateIdString {
	case "head":
		root, err = p.headStateRoot(ctx)
	case "genesis":
		root, err = p.genesisStateRoot(ctx)
	case "finalized":
		root, err = p.finalizedStateRoot(ctx)
	case "justified":
		root, err = p.justifiedStateRoot(ctx)
	default:
		if len(stateId) == 32 {
			root, err = p.stateRootByRoot(ctx, stateId)
		} else {
			slotNumber, parseErr := strconv.ParseUint(stateIdString, 10, 64)
			if parseErr != nil {
				e := NewStateIdParseError(parseErr)
				// ID format does not match any valid options.
				return nil, &e
			}
			root, err = p.stateRootBySlot(ctx, types.Slot(slotNumber))
		}
	}

	return root, err
}

func (p *StateProvider) stateByRoot(ctx context.Context, stateRoot []byte) (state.BeaconState, error) {
	headState, err := p.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state")
	}
	for i, root := range headState.StateRoots() {
		if bytes.Equal(root, stateRoot) {
			blockRoot := headState.BlockRoots()[i]
			return p.StateGenService.StateByRoot(ctx, bytesutil.ToBytes32(blockRoot))
		}
	}

	stateNotFoundErr := NewStateNotFoundError(len(headState.StateRoots()))
	return nil, &stateNotFoundErr
}

// StateBySlot returns the post-state for the requested slot. To generate the state, it uses the
// most recent canonical state prior to the target slot, and all canonical blocks
// between the found state's slot and the target slot.
// process_blocks is applied for all canonical blocks, and process_slots is called for any skipped
// slots, or slots following the most recent canonical block up to and including the target slot.
func (p *StateProvider) StateBySlot(ctx context.Context, target types.Slot) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "statefetcher.StateBySlot")
	defer span.End()

	if target > p.GenesisTimeFetcher.CurrentSlot() {
		return nil, errors.New("requested slot is in the future")
	}
	if target > p.ChainInfoFetcher.HeadSlot() {
		return nil, errors.New("requested slot number is higher than head slot number")
	}

	st, err := p.ReplayerBuilder.ReplayerForSlot(target).ReplayBlocks(ctx)
	if err != nil {
		msg := fmt.Sprintf("error while replaying history to slot=%d", target)
		return nil, errors.Wrap(err, msg)
	}
	return st, nil
}

func (p *StateProvider) headStateRoot(ctx context.Context) ([]byte, error) {
	b, err := p.ChainInfoFetcher.HeadBlock(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head block")
	}
	if err = blocks.BeaconBlockIsNil(b); err != nil {
		return nil, err
	}
	return b.Block().StateRoot(), nil
}

func (p *StateProvider) genesisStateRoot(ctx context.Context) ([]byte, error) {
	b, err := p.BeaconDB.GenesisBlock(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get genesis block")
	}
	if err := blocks.BeaconBlockIsNil(b); err != nil {
		return nil, err
	}
	return b.Block().StateRoot(), nil
}

func (p *StateProvider) finalizedStateRoot(ctx context.Context) ([]byte, error) {
	cp, err := p.BeaconDB.FinalizedCheckpoint(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get finalized checkpoint")
	}
	b, err := p.BeaconDB.Block(ctx, bytesutil.ToBytes32(cp.Root))
	if err != nil {
		return nil, errors.Wrap(err, "could not get finalized block")
	}
	if err := blocks.BeaconBlockIsNil(b); err != nil {
		return nil, err
	}
	return b.Block().StateRoot(), nil
}

func (p *StateProvider) justifiedStateRoot(ctx context.Context) ([]byte, error) {
	cp, err := p.BeaconDB.JustifiedCheckpoint(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get justified checkpoint")
	}
	b, err := p.BeaconDB.Block(ctx, bytesutil.ToBytes32(cp.Root))
	if err != nil {
		return nil, errors.Wrap(err, "could not get justified block")
	}
	if err := blocks.BeaconBlockIsNil(b); err != nil {
		return nil, err
	}
	return b.Block().StateRoot(), nil
}

func (p *StateProvider) stateRootByRoot(ctx context.Context, stateRoot []byte) ([]byte, error) {
	var r [32]byte
	copy(r[:], stateRoot)
	headState, err := p.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state")
	}
	for _, root := range headState.StateRoots() {
		if bytes.Equal(root, r[:]) {
			return r[:], nil
		}
	}

	rootNotFoundErr := NewStateRootNotFoundError(len(headState.StateRoots()))
	return nil, &rootNotFoundErr
}

func (p *StateProvider) stateRootBySlot(ctx context.Context, slot types.Slot) ([]byte, error) {
	currentSlot := p.GenesisTimeFetcher.CurrentSlot()
	if slot > currentSlot {
		return nil, errors.New("slot cannot be in the future")
	}
	blks, err := p.BeaconDB.BlocksBySlot(ctx, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get blocks")
	}
	if len(blks) == 0 {
		return nil, errors.New("no block exists")
	}
	if len(blks) != 1 {
		return nil, errors.New("multiple blocks exist in same slot")
	}
	if blks[0] == nil || blks[0].Block() == nil {
		return nil, errors.New("nil block")
	}
	return blks[0].Block().StateRoot(), nil
}
