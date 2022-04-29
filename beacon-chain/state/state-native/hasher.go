package state_native

import (
	"context"
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"go.opencensus.io/trace"
)

// ComputeFieldRootsWithHasher hashes the provided state and returns its respective field roots.
func ComputeFieldRootsWithHasher(ctx context.Context, state *BeaconState) ([][]byte, error) {
	_, span := trace.StartSpan(ctx, "ComputeFieldRootsWithHasher")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hash.CustomSHA256Hasher()
	var fieldRoots [][]byte
	switch state.version {
	case version.Phase0:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateFieldCount)
	case version.Altair:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateAltairFieldCount)
	case version.Bellatrix:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateBellatrixFieldCount)
	}

	fieldRootIx := 0

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.genesisTime)
	fieldRoots[fieldRootIx] = genesisRoot[:]
	fieldRootIx++

	// Genesis validators root.
	r := [32]byte{}
	copy(r[:], state.genesisValidatorsRoot[:])
	fieldRoots[fieldRootIx] = r[:]
	fieldRootIx++

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.slot))
	fieldRoots[fieldRootIx] = slotRoot[:]
	fieldRootIx++

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[fieldRootIx] = forkHashTreeRoot[:]
	fieldRootIx++

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := stateutil.BlockHeaderRoot(state.latestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[fieldRootIx] = headerHashTreeRoot[:]
	fieldRootIx++

	// BlockRoots array root.
	bRoots := make([][]byte, len(state.blockRoots))
	for i := range bRoots {
		bRoots[i] = state.blockRoots[i][:]
	}
	blockRootsRoot, err := stateutil.ArraysRoot(bRoots, fieldparams.BlockRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[fieldRootIx] = blockRootsRoot[:]
	fieldRootIx++

	// StateRoots array root.
	sRoots := make([][]byte, len(state.stateRoots))
	for i := range sRoots {
		sRoots[i] = state.stateRoots[i][:]
	}
	stateRootsRoot, err := stateutil.ArraysRoot(sRoots, fieldparams.StateRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[fieldRootIx] = stateRootsRoot[:]
	fieldRootIx++

	// HistoricalRoots slice root.
	hRoots := make([][]byte, len(state.historicalRoots))
	for i := range hRoots {
		hRoots[i] = state.historicalRoots[i][:]
	}
	historicalRootsRt, err := ssz.ByteArrayRootWithLimit(hRoots, fieldparams.HistoricalRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[fieldRootIx] = historicalRootsRt[:]
	fieldRootIx++

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := stateutil.Eth1Root(hasher, state.eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[fieldRootIx] = eth1HashTreeRoot[:]
	fieldRootIx++

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := stateutil.Eth1DataVotesRoot(state.eth1DataVotes)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[fieldRootIx] = eth1VotesRoot[:]
	fieldRootIx++

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[fieldRootIx] = eth1DepositBuf[:]
	fieldRootIx++

	// Validators slice root.
	validatorsRoot, err := stateutil.ValidatorRegistryRoot(state.validators)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[fieldRootIx] = validatorsRoot[:]
	fieldRootIx++

	// Balances slice root.
	balancesRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.balances)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[fieldRootIx] = balancesRoot[:]
	fieldRootIx++

	// RandaoMixes array root.
	mixes := make([][]byte, len(state.randaoMixes))
	for i := range mixes {
		mixes[i] = state.randaoMixes[i][:]
	}
	randaoRootsRoot, err := stateutil.ArraysRoot(mixes, fieldparams.RandaoMixesLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[fieldRootIx] = randaoRootsRoot[:]
	fieldRootIx++

	// Slashings array root.
	slashingsRootsRoot, err := ssz.SlashingsRoot(state.slashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[fieldRootIx] = slashingsRootsRoot[:]
	fieldRootIx++

	if state.version == version.Phase0 {
		// PreviousEpochAttestations slice root.
		prevAttsRoot, err := stateutil.EpochAttestationsRoot(state.previousEpochAttestations)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
		}
		fieldRoots[fieldRootIx] = prevAttsRoot[:]
		fieldRootIx++

		// CurrentEpochAttestations slice root.
		currAttsRoot, err := stateutil.EpochAttestationsRoot(state.currentEpochAttestations)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch attestations merkleization")
		}
		fieldRoots[fieldRootIx] = currAttsRoot[:]
		fieldRootIx++
	}

	if state.version == version.Altair || state.version == version.Bellatrix {
		// PreviousEpochParticipation slice root.
		prevParticipationRoot, err := stateutil.ParticipationBitsRoot(state.previousEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch participation merkleization")
		}
		fieldRoots[fieldRootIx] = prevParticipationRoot[:]
		fieldRootIx++

		// CurrentEpochParticipation slice root.
		currParticipationRoot, err := stateutil.ParticipationBitsRoot(state.currentEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch participation merkleization")
		}
		fieldRoots[fieldRootIx] = currParticipationRoot[:]
		fieldRootIx++
	}

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.justificationBits)
	fieldRoots[fieldRootIx] = justifiedBitsRoot[:]
	fieldRootIx++

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(hasher, state.previousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = prevCheckRoot[:]
	fieldRootIx++

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(hasher, state.currentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = currJustRoot[:]
	fieldRootIx++

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(hasher, state.finalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = finalRoot[:]
	fieldRootIx++

	if state.version == version.Altair || state.version == version.Bellatrix {
		// Inactivity scores root.
		inactivityScoresRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.inactivityScores)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute inactivityScoreRoot")
		}
		fieldRoots[fieldRootIx] = inactivityScoresRoot[:]
		fieldRootIx++

		// Current sync committee root.
		currentSyncCommitteeRoot, err := stateutil.SyncCommitteeRoot(state.currentSyncCommittee)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute sync committee merkleization")
		}
		fieldRoots[fieldRootIx] = currentSyncCommitteeRoot[:]
		fieldRootIx++

		// Next sync committee root.
		nextSyncCommitteeRoot, err := stateutil.SyncCommitteeRoot(state.nextSyncCommittee)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute sync committee merkleization")
		}
		fieldRoots[fieldRootIx] = nextSyncCommitteeRoot[:]
		fieldRootIx++
	}

	if state.version == version.Bellatrix {
		// Execution payload root.
		executionPayloadRoot, err := state.latestExecutionPayloadHeader.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		fieldRoots[fieldRootIx] = executionPayloadRoot[:]
	}

	return fieldRoots, nil
}
