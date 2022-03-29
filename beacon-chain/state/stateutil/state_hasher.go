package stateutil

import (
	"context"
	"encoding/binary"

	"github.com/pkg/errors"
	v0 "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v0"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"go.opencensus.io/trace"
)

// TODO: Better doc
// ComputeFieldRootsWithHasher hashes the provided state and returns its respective field roots.
func ComputeFieldRootsWithHasher(ctx context.Context, state *v0.BeaconState) ([][]byte, error) {
	_, span := trace.StartSpan(ctx, "ComputeFieldRootsWithHasher")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hash.CustomSHA256Hasher()
	var fieldRoots [][]byte
	switch state.Version() {
	case int(v0.Phase0):
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateFieldCount)
	case int(v0.Altair):
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateAltairFieldCount)
	case int(v0.Bellatrix):
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateBellatrixFieldCount)
	}

	fieldRootIx := 0

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.GenesisTime())
	fieldRoots[fieldRootIx] = genesisRoot[:]
	fieldRootIx++

	// Genesis validators root.
	r := [32]byte{}
	copy(r[:], state.GenesisValidatorsRoot())
	fieldRoots[fieldRootIx] = r[:]
	fieldRootIx++

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.Slot()))
	fieldRoots[fieldRootIx] = slotRoot[:]
	fieldRootIx++

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.Fork())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[fieldRootIx] = forkHashTreeRoot[:]
	fieldRootIx++

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := BlockHeaderRoot(state.LatestBlockHeader())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[fieldRootIx] = headerHashTreeRoot[:]
	fieldRootIx++

	// BlockRoots array root.
	blockRootsRoot, err := arraysRoot(state.BlockRoots(), fieldparams.BlockRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[fieldRootIx] = blockRootsRoot[:]
	fieldRootIx++

	// StateRoots array root.
	stateRootsRoot, err := arraysRoot(state.StateRoots(), fieldparams.StateRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[fieldRootIx] = stateRootsRoot[:]
	fieldRootIx++

	// HistoricalRoots slice root.
	historicalRootsRt, err := ssz.ByteArrayRootWithLimit(state.HistoricalRoots(), fieldparams.HistoricalRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[fieldRootIx] = historicalRootsRt[:]
	fieldRootIx++

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := Eth1Root(hasher, state.Eth1Data())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[fieldRootIx] = eth1HashTreeRoot[:]
	fieldRootIx++

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := eth1DataVotesRoot(state.Eth1DataVotes())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[fieldRootIx] = eth1VotesRoot[:]
	fieldRootIx++

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.Eth1DepositIndex())
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[fieldRootIx] = eth1DepositBuf[:]
	fieldRootIx++

	// Validators slice root.
	validatorsRoot, err := validatorRegistryRoot(state.Validators())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[fieldRootIx] = validatorsRoot[:]
	fieldRootIx++

	// Balances slice root.
	balancesRoot, err := Uint64ListRootWithRegistryLimit(state.Balances())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[fieldRootIx] = balancesRoot[:]
	fieldRootIx++

	// RandaoMixes array root.
	randaoRootsRoot, err := arraysRoot(state.RandaoMixes(), fieldparams.RandaoMixesLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[fieldRootIx] = randaoRootsRoot[:]
	fieldRootIx++

	// Slashings array root.
	slashingsRootsRoot, err := ssz.SlashingsRoot(state.Slashings())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[fieldRootIx] = slashingsRootsRoot[:]
	fieldRootIx++

	if state.Version() == version.Phase0 {
		// PreviousEpochAttestations slice root.
		prevEpochAtts, err := state.PreviousEpochAttestations()
		if err != nil {
			return nil, errors.Wrap(err, "could not get previous epoch attestations")
		}
		prevAttsRoot, err := epochAttestationsRoot(prevEpochAtts)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
		}
		fieldRoots[fieldRootIx] = prevAttsRoot[:]
		fieldRootIx++

		// CurrentEpochAttestations slice root.
		currEpochAtts, err := state.CurrentEpochAttestations()
		if err != nil {
			return nil, errors.Wrap(err, "could not get current epoch attestations")
		}
		currAttsRoot, err := epochAttestationsRoot(currEpochAtts)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch attestations merkleization")
		}
		fieldRoots[fieldRootIx] = currAttsRoot[:]
		fieldRootIx++
	}

	if state.Version() == version.Altair || state.Version() == version.Bellatrix {
		// PreviousEpochParticipation slice root.
		prevEpochParticipation, err := state.PreviousEpochParticipation()
		if err != nil {
			return nil, errors.Wrap(err, "could not get previous epoch participation")
		}
		prevParticipationRoot, err := ParticipationBitsRoot(prevEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch participation merkleization")
		}
		fieldRoots[fieldRootIx] = prevParticipationRoot[:]
		fieldRootIx++

		// CurrentEpochParticipation slice root.
		currEpochParticipation, err := state.CurrentEpochParticipation()
		if err != nil {
			return nil, errors.Wrap(err, "could not get current epoch participation")
		}
		currParticipationRoot, err := ParticipationBitsRoot(currEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch participation merkleization")
		}
		fieldRoots[fieldRootIx] = currParticipationRoot[:]
		fieldRootIx++
	}

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits())
	fieldRoots[fieldRootIx] = justifiedBitsRoot[:]
	fieldRootIx++

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(hasher, state.PreviousJustifiedCheckpoint())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = prevCheckRoot[:]
	fieldRootIx++

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(hasher, state.CurrentJustifiedCheckpoint())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = currJustRoot[:]
	fieldRootIx++

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(hasher, state.FinalizedCheckpoint())
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = finalRoot[:]
	fieldRootIx++

	if state.Version() == version.Altair || state.Version() == version.Bellatrix {
		// Current sync committee root.
		currSyncCommittee, err := state.CurrentSyncCommittee()
		if err != nil {
			return nil, errors.Wrap(err, "could not get current sync committee")
		}
		currentSyncCommitteeRoot, err := SyncCommitteeRoot(currSyncCommittee)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute sync committee merkleization")
		}
		fieldRoots[fieldRootIx] = currentSyncCommitteeRoot[:]
		fieldRootIx++

		// Next sync committee root.
		nextSyncCommittee, err := state.NextSyncCommittee()
		if err != nil {
			return nil, errors.Wrap(err, "could not get next sync committee")
		}
		nextSyncCommitteeRoot, err := SyncCommitteeRoot(nextSyncCommittee)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute sync committee merkleization")
		}
		fieldRoots[fieldRootIx] = nextSyncCommitteeRoot[:]
		fieldRootIx++
	}

	if state.Version() == version.Bellatrix {
		// Execution payload root.
		header, err := state.LatestExecutionPayloadHeader()
		if err != nil {
			return nil, errors.Wrap(err, "could not get latest execution payload header")
		}
		executionPayloadRoot, err := header.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		fieldRoots[fieldRootIx] = executionPayloadRoot[:]
		fieldRootIx++
	}

	return fieldRoots, nil
}

// ComputeFieldRootsWithHasherPhase0 hashes the provided phase 0 state and returns its respective field roots.
func ComputeFieldRootsWithHasherPhase0(ctx context.Context, state *ethpb.BeaconState) ([][]byte, error) {
	_, span := trace.StartSpan(ctx, "ComputeFieldRootsWithHasherPhase0")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hash.CustomSHA256Hasher()
	fieldRoots := make([][]byte, params.BeaconConfig().BeaconStateFieldCount)

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.GenesisTime)
	fieldRoots[0] = genesisRoot[:]

	// Genesis validators root.
	r := [32]byte{}
	copy(r[:], state.GenesisValidatorsRoot)
	fieldRoots[1] = r[:]

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.Slot))
	fieldRoots[2] = slotRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.Fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[3] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := BlockHeaderRoot(state.LatestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[4] = headerHashTreeRoot[:]

	// BlockRoots array root.
	blockRootsRoot, err := arraysRoot(state.BlockRoots, fieldparams.BlockRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[5] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := arraysRoot(state.StateRoots, fieldparams.StateRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[6] = stateRootsRoot[:]

	// HistoricalRoots slice root.
	historicalRootsRt, err := ssz.ByteArrayRootWithLimit(state.HistoricalRoots, fieldparams.HistoricalRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[7] = historicalRootsRt[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := Eth1Root(hasher, state.Eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[8] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := eth1DataVotesRoot(state.Eth1DataVotes)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[9] = eth1VotesRoot[:]

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.Eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[10] = eth1DepositBuf[:]

	// Validators slice root.
	validatorsRoot, err := validatorRegistryRoot(state.Validators)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[11] = validatorsRoot[:]

	// Balances slice root.
	balancesRoot, err := Uint64ListRootWithRegistryLimit(state.Balances)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[12] = balancesRoot[:]

	// RandaoMixes array root.
	randaoRootsRoot, err := arraysRoot(state.RandaoMixes, fieldparams.RandaoMixesLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[13] = randaoRootsRoot[:]

	// Slashings array root.
	slashingsRootsRoot, err := ssz.SlashingsRoot(state.Slashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[14] = slashingsRootsRoot[:]

	// PreviousEpochAttestations slice root.
	prevAttsRoot, err := epochAttestationsRoot(state.PreviousEpochAttestations)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
	}
	fieldRoots[15] = prevAttsRoot[:]

	// CurrentEpochAttestations slice root.
	currAttsRoot, err := epochAttestationsRoot(state.CurrentEpochAttestations)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current epoch attestations merkleization")
	}
	fieldRoots[16] = currAttsRoot[:]

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits)
	fieldRoots[17] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(hasher, state.PreviousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[18] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(hasher, state.CurrentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[19] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(hasher, state.FinalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[20] = finalRoot[:]
	return fieldRoots, nil
}

// ComputeFieldRootsWithHasherAltair hashes the provided altair state and returns its respective field roots.
func ComputeFieldRootsWithHasherAltair(ctx context.Context, state *ethpb.BeaconStateAltair) ([][]byte, error) {
	_, span := trace.StartSpan(ctx, "ComputeFieldRootsWithHasherAltair")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hash.CustomSHA256Hasher()
	fieldRoots := make([][]byte, params.BeaconConfig().BeaconStateAltairFieldCount)

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.GenesisTime)
	fieldRoots[0] = genesisRoot[:]

	// Genesis validators root.
	r := [32]byte{}
	copy(r[:], state.GenesisValidatorsRoot)
	fieldRoots[1] = r[:]

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.Slot))
	fieldRoots[2] = slotRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.Fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[3] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := BlockHeaderRoot(state.LatestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[4] = headerHashTreeRoot[:]

	// BlockRoots array root.
	blockRootsRoot, err := arraysRoot(state.BlockRoots, fieldparams.BlockRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[5] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := arraysRoot(state.StateRoots, fieldparams.StateRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[6] = stateRootsRoot[:]

	// HistoricalRoots slice root.
	historicalRootsRt, err := ssz.ByteArrayRootWithLimit(state.HistoricalRoots, fieldparams.HistoricalRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[7] = historicalRootsRt[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := Eth1Root(hasher, state.Eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[8] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := eth1DataVotesRoot(state.Eth1DataVotes)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[9] = eth1VotesRoot[:]

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.Eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[10] = eth1DepositBuf[:]

	// Validators slice root.
	validatorsRoot, err := validatorRegistryRoot(state.Validators)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[11] = validatorsRoot[:]

	// Balances slice root.
	balancesRoot, err := Uint64ListRootWithRegistryLimit(state.Balances)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[12] = balancesRoot[:]

	// RandaoMixes array root.
	randaoRootsRoot, err := arraysRoot(state.RandaoMixes, fieldparams.RandaoMixesLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[13] = randaoRootsRoot[:]

	// Slashings array root.
	slashingsRootsRoot, err := ssz.SlashingsRoot(state.Slashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[14] = slashingsRootsRoot[:]

	// PreviousEpochParticipation slice root.
	prevParticipationRoot, err := ParticipationBitsRoot(state.PreviousEpochParticipation)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous epoch participation merkleization")
	}
	fieldRoots[15] = prevParticipationRoot[:]

	// CurrentEpochParticipation slice root.
	currParticipationRoot, err := ParticipationBitsRoot(state.CurrentEpochParticipation)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current epoch participation merkleization")
	}
	fieldRoots[16] = currParticipationRoot[:]

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits)
	fieldRoots[17] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(hasher, state.PreviousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[18] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(hasher, state.CurrentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[19] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(hasher, state.FinalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[20] = finalRoot[:]

	// Inactivity scores root.
	inactivityScoresRoot, err := Uint64ListRootWithRegistryLimit(state.InactivityScores)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute inactivityScoreRoot")
	}
	fieldRoots[21] = inactivityScoresRoot[:]

	// Current sync committee root.
	currentSyncCommitteeRoot, err := SyncCommitteeRoot(state.CurrentSyncCommittee)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute sync committee merkleization")
	}
	fieldRoots[22] = currentSyncCommitteeRoot[:]

	// Next sync committee root.
	nextSyncCommitteeRoot, err := SyncCommitteeRoot(state.NextSyncCommittee)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute sync committee merkleization")
	}
	fieldRoots[23] = nextSyncCommitteeRoot[:]

	return fieldRoots, nil
}

// ComputeFieldRootsWithHasherBellatrix hashes the provided bellatrix state and returns its respective field roots.
func ComputeFieldRootsWithHasherBellatrix(ctx context.Context, state *ethpb.BeaconStateBellatrix) ([][]byte, error) {
	_, span := trace.StartSpan(ctx, "ComputeFieldRootsWithHasherBellatrix")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hash.CustomSHA256Hasher()
	fieldRoots := make([][]byte, params.BeaconConfig().BeaconStateBellatrixFieldCount)

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.GenesisTime)
	fieldRoots[0] = genesisRoot[:]

	// Genesis validators root.
	r := [32]byte{}
	copy(r[:], state.GenesisValidatorsRoot)
	fieldRoots[1] = r[:]

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.Slot))
	fieldRoots[2] = slotRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.Fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[3] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := BlockHeaderRoot(state.LatestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[4] = headerHashTreeRoot[:]

	// BlockRoots array root.
	blockRootsRoot, err := arraysRoot(state.BlockRoots, fieldparams.BlockRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[5] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := arraysRoot(state.StateRoots, fieldparams.StateRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[6] = stateRootsRoot[:]

	// HistoricalRoots slice root.
	historicalRootsRt, err := ssz.ByteArrayRootWithLimit(state.HistoricalRoots, fieldparams.HistoricalRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[7] = historicalRootsRt[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := Eth1Root(hasher, state.Eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[8] = eth1HashTreeRoot[:]

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := eth1DataVotesRoot(state.Eth1DataVotes)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[9] = eth1VotesRoot[:]

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.Eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[10] = eth1DepositBuf[:]

	// Validators slice root.
	validatorsRoot, err := validatorRegistryRoot(state.Validators)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[11] = validatorsRoot[:]

	// Balances slice root.
	balancesRoot, err := Uint64ListRootWithRegistryLimit(state.Balances)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[12] = balancesRoot[:]

	// RandaoMixes array root.
	randaoRootsRoot, err := arraysRoot(state.RandaoMixes, fieldparams.RandaoMixesLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[13] = randaoRootsRoot[:]

	// Slashings array root.
	slashingsRootsRoot, err := ssz.SlashingsRoot(state.Slashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[14] = slashingsRootsRoot[:]

	// PreviousEpochParticipation slice root.
	prevParticipationRoot, err := ParticipationBitsRoot(state.PreviousEpochParticipation)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous epoch participation merkleization")
	}
	fieldRoots[15] = prevParticipationRoot[:]

	// CurrentEpochParticipation slice root.
	currParticipationRoot, err := ParticipationBitsRoot(state.CurrentEpochParticipation)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current epoch participation merkleization")
	}
	fieldRoots[16] = currParticipationRoot[:]

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits)
	fieldRoots[17] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(hasher, state.PreviousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[18] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(hasher, state.CurrentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[19] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(hasher, state.FinalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[20] = finalRoot[:]

	// Inactivity scores root.
	inactivityScoresRoot, err := Uint64ListRootWithRegistryLimit(state.InactivityScores)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute inactivityScoreRoot")
	}
	fieldRoots[21] = inactivityScoresRoot[:]

	// Current sync committee root.
	currentSyncCommitteeRoot, err := SyncCommitteeRoot(state.CurrentSyncCommittee)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute sync committee merkleization")
	}
	fieldRoots[22] = currentSyncCommitteeRoot[:]

	// Next sync committee root.
	nextSyncCommitteeRoot, err := SyncCommitteeRoot(state.NextSyncCommittee)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute sync committee merkleization")
	}
	fieldRoots[23] = nextSyncCommitteeRoot[:]

	// Execution payload root.
	executionPayloadRoot, err := state.LatestExecutionPayloadHeader.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	fieldRoots[24] = executionPayloadRoot[:]

	return fieldRoots, nil
}
