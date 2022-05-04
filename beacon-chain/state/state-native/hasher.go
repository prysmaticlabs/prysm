package state_native

import (
	"context"
	"encoding/binary"

	"github.com/pkg/errors"
	nativetypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/types"
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

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.genesisTime)
	fieldRoots[nativetypes.GenesisTime.RealPosition(state)] = genesisRoot[:]

	// Genesis validators root.
	r := [32]byte{}
	copy(r[:], state.genesisValidatorsRoot[:])
	fieldRoots[nativetypes.GenesisValidatorsRoot.RealPosition(state)] = r[:]

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.slot))
	fieldRoots[nativetypes.Slot.RealPosition(state)] = slotRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[nativetypes.Fork.RealPosition(state)] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := stateutil.BlockHeaderRoot(state.latestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[nativetypes.LatestBlockHeader.RealPosition(state)] = headerHashTreeRoot[:]

	// BlockRoots array root.
	bRoots := make([][]byte, len(state.blockRoots))
	for i := range bRoots {
		bRoots[i] = state.blockRoots[i][:]
	}
	blockRootsRoot, err := stateutil.ArraysRoot(bRoots, fieldparams.BlockRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[nativetypes.BlockRoots.RealPosition(state)] = blockRootsRoot[:]

	// StateRoots array root.
	sRoots := make([][]byte, len(state.stateRoots))
	for i := range sRoots {
		sRoots[i] = state.stateRoots[i][:]
	}
	stateRootsRoot, err := stateutil.ArraysRoot(sRoots, fieldparams.StateRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[nativetypes.StateRoots.RealPosition(state)] = stateRootsRoot[:]

	// HistoricalRoots slice root.
	hRoots := make([][]byte, len(state.historicalRoots))
	for i := range hRoots {
		hRoots[i] = state.historicalRoots[i][:]
	}
	historicalRootsRt, err := ssz.ByteArrayRootWithLimit(hRoots, fieldparams.HistoricalRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[nativetypes.HistoricalRoots.RealPosition(state)] = historicalRootsRt[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := stateutil.Eth1Root(hasher, state.eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[nativetypes.Eth1Data.RealPosition(state)] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := stateutil.Eth1DataVotesRoot(state.eth1DataVotes)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[nativetypes.Eth1DataVotes.RealPosition(state)] = eth1VotesRoot[:]

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[nativetypes.Eth1DepositIndex.RealPosition(state)] = eth1DepositBuf[:]

	// Validators slice root.
	validatorsRoot, err := stateutil.ValidatorRegistryRoot(state.validators)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[nativetypes.Validators.RealPosition(state)] = validatorsRoot[:]

	// Balances slice root.
	balancesRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.balances)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[nativetypes.Balances.RealPosition(state)] = balancesRoot[:]

	// RandaoMixes array root.
	mixes := make([][]byte, len(state.randaoMixes))
	for i := range mixes {
		mixes[i] = state.randaoMixes[i][:]
	}
	randaoRootsRoot, err := stateutil.ArraysRoot(mixes, fieldparams.RandaoMixesLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[nativetypes.RandaoMixes.RealPosition(state)] = randaoRootsRoot[:]

	// Slashings array root.
	slashingsRootsRoot, err := ssz.SlashingsRoot(state.slashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[nativetypes.Slashings.RealPosition(state)] = slashingsRootsRoot[:]

	if state.version == version.Phase0 {
		// PreviousEpochAttestations slice root.
		prevAttsRoot, err := stateutil.EpochAttestationsRoot(state.previousEpochAttestations)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
		}
		fieldRoots[nativetypes.PreviousEpochAttestations.RealPosition(state)] = prevAttsRoot[:]

		// CurrentEpochAttestations slice root.
		currAttsRoot, err := stateutil.EpochAttestationsRoot(state.currentEpochAttestations)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch attestations merkleization")
		}
		fieldRoots[nativetypes.CurrentEpochAttestations.RealPosition(state)] = currAttsRoot[:]
	}

	if state.version == version.Altair || state.version == version.Bellatrix {
		// PreviousEpochParticipation slice root.
		prevParticipationRoot, err := stateutil.ParticipationBitsRoot(state.previousEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch participation merkleization")
		}
		fieldRoots[nativetypes.PreviousEpochParticipationBits.RealPosition(state)] = prevParticipationRoot[:]

		// CurrentEpochParticipation slice root.
		currParticipationRoot, err := stateutil.ParticipationBitsRoot(state.currentEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch participation merkleization")
		}
		fieldRoots[nativetypes.CurrentEpochParticipationBits.RealPosition(state)] = currParticipationRoot[:]
	}

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.justificationBits)
	fieldRoots[nativetypes.JustificationBits.RealPosition(state)] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(hasher, state.previousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[nativetypes.PreviousJustifiedCheckpoint.RealPosition(state)] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(hasher, state.currentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[nativetypes.CurrentJustifiedCheckpoint.RealPosition(state)] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(hasher, state.finalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[nativetypes.FinalizedCheckpoint.RealPosition(state)] = finalRoot[:]

	if state.version == version.Altair || state.version == version.Bellatrix {
		// Inactivity scores root.
		inactivityScoresRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.inactivityScores)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute inactivityScoreRoot")
		}
		fieldRoots[nativetypes.InactivityScores.RealPosition(state)] = inactivityScoresRoot[:]

		// Current sync committee root.
		currentSyncCommitteeRoot, err := stateutil.SyncCommitteeRoot(state.currentSyncCommittee)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute sync committee merkleization")
		}
		fieldRoots[nativetypes.CurrentSyncCommittee.RealPosition(state)] = currentSyncCommitteeRoot[:]

		// Next sync committee root.
		nextSyncCommitteeRoot, err := stateutil.SyncCommitteeRoot(state.nextSyncCommittee)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute sync committee merkleization")
		}
		fieldRoots[nativetypes.NextSyncCommittee.RealPosition(state)] = nextSyncCommitteeRoot[:]
	}

	if state.version == version.Bellatrix {
		// Execution payload root.
		executionPayloadRoot, err := state.latestExecutionPayloadHeader.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		fieldRoots[nativetypes.LatestExecutionPayloadHeader.RealPosition(state)] = executionPayloadRoot[:]
	}

	return fieldRoots, nil
}
