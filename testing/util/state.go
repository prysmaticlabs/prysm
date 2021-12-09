package util

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// FillRootsNaturalOpt is meant to be used as an option when calling NewBeaconState.
// It fills state and block roots with hex representations of natural numbers starting with 0.
// Example: 16 becomes 0x00...0f.
func FillRootsNaturalOpt(state state.BeaconState) error {
	rootsLen := params.BeaconConfig().SlotsPerHistoricalRoot
	roots := make([][]byte, rootsLen)
	for i := types.Slot(0); i < rootsLen; i++ {
		roots[i] = make([]byte, 32)
	}
	for j := 0; j < len(roots); j++ {
		// Remove '0x' prefix and left-pad '0' to have 64 chars in total.
		s := fmt.Sprintf("%064s", hexutil.EncodeUint64(uint64(j))[2:])
		h, err := hexutil.Decode("0x" + s)
		if err != nil {
			return err
		}
		roots[j] = h
	}
	var stateRoots [customtypes.StateRootsSize][32]byte
	for i := range stateRoots {
		stateRoots[i] = bytesutil.ToBytes32(roots[i])
	}
	if err := state.SetStateRoots(&stateRoots); err != nil {
		return errors.Wrap(err, "could not set state roots")
	}
	var blockRoots [customtypes.BlockRootsSize][32]byte
	for i := range blockRoots {
		blockRoots[i] = bytesutil.ToBytes32(roots[i])
	}
	if err := state.SetBlockRoots(&blockRoots); err != nil {
		return errors.Wrap(err, "could not set block roots")
	}
	return nil
}

// NewBeaconState creates a beacon state with minimum marshalable fields.
func NewBeaconState(options ...func(beaconState state.BeaconState) error) (*v1.BeaconState, error) {
	var st, err = v1.Initialize()
	if err != nil {
		return nil, err
	}

	if err = st.SetFork(&ethpb.Fork{
		PreviousVersion: make([]byte, 4),
		CurrentVersion:  make([]byte, 4),
	}); err != nil {
		return nil, errors.Wrap(err, "could not set fork")
	}
	if err = st.SetLatestBlockHeader(HydrateBeaconHeader(&ethpb.BeaconBlockHeader{})); err != nil {
		return nil, errors.Wrap(err, "could not set latest block header")
	}
	if err = st.SetHistoricalRoots([][32]byte{}); err != nil {
		return nil, errors.Wrap(err, "could not set historical roots")
	}
	if err = st.SetBlockRoots(&[customtypes.BlockRootsSize][32]byte{}); err != nil {
		return nil, errors.Wrap(err, "could not set block roots")
	}
	if err = st.SetStateRoots(&[customtypes.StateRootsSize][32]byte{}); err != nil {
		return nil, errors.Wrap(err, "could not set state roots")
	}
	if err = st.SetEth1Data(&ethpb.Eth1Data{
		DepositRoot: make([]byte, 32),
		BlockHash:   make([]byte, 32),
	}); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data")
	}
	if err = st.SetEth1DataVotes(make([]*ethpb.Eth1Data, 0)); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data votes")
	}
	if err = st.SetValidators(make([]*ethpb.Validator, 0)); err != nil {
		return nil, errors.Wrap(err, "could not set validators")
	}
	if err = st.SetRandaoMixes(&[customtypes.RandaoMixesSize][32]byte{}); err != nil {
		return nil, errors.Wrap(err, "could not set randao mixes")
	}
	if err = st.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)); err != nil {
		return nil, errors.Wrap(err, "could not set slashings")
	}
	if err = st.SetJustificationBits(bitfield.Bitvector4{0x0}); err != nil {
		return nil, errors.Wrap(err, "could not set justification bits")
	}
	if err = st.SetPreviousJustifiedCheckpoint(&ethpb.Checkpoint{Root: make([]byte, 32)}); err != nil {
		return nil, errors.Wrap(err, "could not set previous justified checkpoint")
	}
	if err = st.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{Root: make([]byte, 32)}); err != nil {
		return nil, errors.Wrap(err, "could not set current justified checkpoint")
	}
	if err = st.SetFinalizedCheckpoint(&ethpb.Checkpoint{Root: make([]byte, 32)}); err != nil {
		return nil, errors.Wrap(err, "could not set finalized checkpoint")
	}

	for _, opt := range options {
		err = opt(st)
		if err != nil {
			return nil, err
		}
	}

	return st.Copy().(*v1.BeaconState), nil
}

// SSZ will fill 2D byte slices with their respective values, so we must fill these in too for round
// trip testing.
func filledByteSlice2D(length, innerLen uint64) [][]byte {
	b := make([][]byte, length)
	for i := uint64(0); i < length; i++ {
		b[i] = make([]byte, innerLen)
	}
	return b
}
