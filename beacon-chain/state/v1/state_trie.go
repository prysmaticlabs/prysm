package v1

import (
	"context"
	"runtime"
	"sort"
	"sync"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

var (
	stateCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_state_count",
		Help: "Count the number of active beacon state objects.",
	})
)

// InitializeFromProto the beacon state from a protobuf representation.
func InitializeFromProto(st *ethpb.BeaconState) (*BeaconState, error) {
	return InitializeFromProtoUnsafe(proto.Clone(st).(*ethpb.BeaconState))
}

// InitializeFromProtoUnsafe directly uses the beacon state protobuf pointer
// and sets it as the inner state of the BeaconState type.
func InitializeFromProtoUnsafe(st *ethpb.BeaconState) (*BeaconState, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	fieldCount := params.BeaconConfig().BeaconStateFieldCount
	b := &BeaconState{
		state:                 st,
		dirtyFields:           make(map[fieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[fieldIndex][]uint64, fieldCount),
		stateFieldLeaves:      make(map[fieldIndex]*FieldTrie, fieldCount),
		sharedFieldReferences: make(map[fieldIndex]*stateutil.Reference, 10),
		rebuildTrie:           make(map[fieldIndex]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	for i := 0; i < fieldCount; i++ {
		b.dirtyFields[fieldIndex(i)] = true
		b.rebuildTrie[fieldIndex(i)] = true
		b.dirtyIndices[fieldIndex(i)] = []uint64{}
		b.stateFieldLeaves[fieldIndex(i)] = &FieldTrie{
			field:     fieldIndex(i),
			reference: stateutil.NewRef(1),
			RWMutex:   new(sync.RWMutex),
		}
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[randaoMixes] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[blockRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[previousEpochAttestations] = stateutil.NewRef(1)
	b.sharedFieldReferences[currentEpochAttestations] = stateutil.NewRef(1)
	b.sharedFieldReferences[slashings] = stateutil.NewRef(1)
	b.sharedFieldReferences[eth1DataVotes] = stateutil.NewRef(1)
	b.sharedFieldReferences[validators] = stateutil.NewRef(1)
	b.sharedFieldReferences[balances] = stateutil.NewRef(1)
	b.sharedFieldReferences[historicalRoots] = stateutil.NewRef(1)

	stateCount.Inc()
	return b, nil
}

// Copy returns a deep copy of the beacon state.
func (b *BeaconState) Copy() state.BeaconState {
	if !b.hasInnerState() {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()
	fieldCount := params.BeaconConfig().BeaconStateFieldCount
	dst := &BeaconState{
		state: &ethpb.BeaconState{
			// Primitive types, safe to copy.
			GenesisTime:      b.state.GenesisTime,
			Slot:             b.state.Slot,
			Eth1DepositIndex: b.state.Eth1DepositIndex,

			// Large arrays, infrequently changed, constant size.
			RandaoMixes:               b.state.RandaoMixes,
			StateRoots:                b.state.StateRoots,
			BlockRoots:                b.state.BlockRoots,
			PreviousEpochAttestations: b.state.PreviousEpochAttestations,
			CurrentEpochAttestations:  b.state.CurrentEpochAttestations,
			Slashings:                 b.state.Slashings,
			Eth1DataVotes:             b.state.Eth1DataVotes,

			// Large arrays, increases over time.
			Validators:      b.state.Validators,
			Balances:        b.state.Balances,
			HistoricalRoots: b.state.HistoricalRoots,

			// Everything else, too small to be concerned about, constant size.
			Fork:                        b.fork(),
			LatestBlockHeader:           b.latestBlockHeader(),
			Eth1Data:                    b.eth1Data(),
			JustificationBits:           b.justificationBits(),
			PreviousJustifiedCheckpoint: b.previousJustifiedCheckpoint(),
			CurrentJustifiedCheckpoint:  b.currentJustifiedCheckpoint(),
			FinalizedCheckpoint:         b.finalizedCheckpoint(),
			GenesisValidatorsRoot:       b.genesisValidatorRoot(),
		},
		dirtyFields:           make(map[fieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[fieldIndex][]uint64, fieldCount),
		rebuildTrie:           make(map[fieldIndex]bool, fieldCount),
		sharedFieldReferences: make(map[fieldIndex]*stateutil.Reference, 10),
		stateFieldLeaves:      make(map[fieldIndex]*FieldTrie, fieldCount),

		// Copy on write validator index map.
		valMapHandler: b.valMapHandler,
	}

	for field, ref := range b.sharedFieldReferences {
		ref.AddRef()
		dst.sharedFieldReferences[field] = ref
	}

	// Increment ref for validator map
	b.valMapHandler.AddRef()

	for i := range b.dirtyFields {
		dst.dirtyFields[i] = true
	}

	for i := range b.dirtyIndices {
		indices := make([]uint64, len(b.dirtyIndices[i]))
		copy(indices, b.dirtyIndices[i])
		dst.dirtyIndices[i] = indices
	}

	for i := range b.rebuildTrie {
		dst.rebuildTrie[i] = true
	}

	for fldIdx, fieldTrie := range b.stateFieldLeaves {
		dst.stateFieldLeaves[fldIdx] = fieldTrie
		if fieldTrie.reference != nil {
			fieldTrie.Lock()
			fieldTrie.reference.AddRef()
			fieldTrie.Unlock()
		}
	}

	if b.merkleLayers != nil {
		dst.merkleLayers = make([][][]byte, len(b.merkleLayers))
		for i, layer := range b.merkleLayers {
			dst.merkleLayers[i] = make([][]byte, len(layer))
			for j, content := range layer {
				dst.merkleLayers[i][j] = make([]byte, len(content))
				copy(dst.merkleLayers[i][j], content)
			}
		}
	}

	stateCount.Inc()
	// Finalizer runs when dst is being destroyed in garbage collection.
	runtime.SetFinalizer(dst, func(b *BeaconState) {
		for field, v := range b.sharedFieldReferences {
			v.MinusRef()
			if b.stateFieldLeaves[field].reference != nil {
				b.stateFieldLeaves[field].reference.MinusRef()
			}

		}
		for i := 0; i < fieldCount; i++ {
			field := fieldIndex(i)
			delete(b.stateFieldLeaves, field)
			delete(b.dirtyIndices, field)
			delete(b.dirtyFields, field)
			delete(b.sharedFieldReferences, field)
			delete(b.stateFieldLeaves, field)
		}
		stateCount.Sub(1)
	})
	return dst
}

// HashTreeRoot of the beacon state retrieves the Merkle root of the trie
// representation of the beacon state based on the Ethereum Simple Serialize specification.
func (b *BeaconState) HashTreeRoot(ctx context.Context) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconState.HashTreeRoot")
	defer span.End()

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.merkleLayers == nil || len(b.merkleLayers) == 0 {
		fieldRoots, err := computeFieldRoots(ctx, b.state)
		if err != nil {
			return [32]byte{}, err
		}
		layers := stateutil.Merkleize(fieldRoots)
		b.merkleLayers = layers
		b.dirtyFields = make(map[fieldIndex]bool, params.BeaconConfig().BeaconStateFieldCount)
	}

	for field := range b.dirtyFields {
		root, err := b.rootSelector(ctx, field)
		if err != nil {
			return [32]byte{}, err
		}
		b.merkleLayers[0][field] = root[:]
		b.recomputeRoot(int(field))
		delete(b.dirtyFields, field)
	}
	return bytesutil.ToBytes32(b.merkleLayers[len(b.merkleLayers)-1][0]), nil
}

// ToProto returns a protobuf *v1.BeaconState representation of the state.
func (b *BeaconState) ToProto() (*v1.BeaconState, error) {
	sourceFork := b.Fork()
	sourceLatestBlockHeader := b.LatestBlockHeader()
	sourceEth1Data := b.Eth1Data()
	sourceEth1DataVotes := b.Eth1DataVotes()
	sourceValidators := b.Validators()
	sourcePrevEpochAtts, err := b.PreviousEpochAttestations()
	if err != nil {
		return nil, errors.Wrap(err, "could not get previous epoch attestations")
	}
	sourceCurrEpochAtts, err := b.CurrentEpochAttestations()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current epoch attestations")
	}
	sourcePrevJustifiedCheckpoint := b.PreviousJustifiedCheckpoint()
	sourceCurrJustifiedCheckpoint := b.CurrentJustifiedCheckpoint()
	sourceFinalizedCheckpoint := b.FinalizedCheckpoint()

	resultEth1DataVotes := make([]*v1.Eth1Data, len(sourceEth1DataVotes))
	for i, vote := range sourceEth1DataVotes {
		resultEth1DataVotes[i] = &v1.Eth1Data{
			DepositRoot:  vote.DepositRoot,
			DepositCount: vote.DepositCount,
			BlockHash:    vote.BlockHash,
		}
	}
	resultValidators := make([]*v1.Validator, len(sourceValidators))
	for i, validator := range sourceValidators {
		resultValidators[i] = &v1.Validator{
			Pubkey:                     validator.PublicKey,
			WithdrawalCredentials:      validator.WithdrawalCredentials,
			EffectiveBalance:           validator.EffectiveBalance,
			Slashed:                    validator.Slashed,
			ActivationEligibilityEpoch: validator.ActivationEligibilityEpoch,
			ActivationEpoch:            validator.ActivationEpoch,
			ExitEpoch:                  validator.ExitEpoch,
			WithdrawableEpoch:          validator.WithdrawableEpoch,
		}
	}
	resultPrevEpochAtts := make([]*v1.PendingAttestation, len(sourcePrevEpochAtts))
	for i, att := range sourcePrevEpochAtts {
		data := att.Data
		resultPrevEpochAtts[i] = &v1.PendingAttestation{
			AggregationBits: att.AggregationBits,
			Data: &v1.AttestationData{
				Slot:            data.Slot,
				Index:           data.CommitteeIndex,
				BeaconBlockRoot: data.BeaconBlockRoot,
				Source: &v1.Checkpoint{
					Epoch: data.Source.Epoch,
					Root:  data.Source.Root,
				},
				Target: &v1.Checkpoint{
					Epoch: data.Target.Epoch,
					Root:  data.Target.Root,
				},
			},
			InclusionDelay: att.InclusionDelay,
			ProposerIndex:  att.ProposerIndex,
		}
	}
	resultCurrEpochAtts := make([]*v1.PendingAttestation, len(sourceCurrEpochAtts))
	for i, att := range sourceCurrEpochAtts {
		data := att.Data
		resultCurrEpochAtts[i] = &v1.PendingAttestation{
			AggregationBits: att.AggregationBits,
			Data: &v1.AttestationData{
				Slot:            data.Slot,
				Index:           data.CommitteeIndex,
				BeaconBlockRoot: data.BeaconBlockRoot,
				Source: &v1.Checkpoint{
					Epoch: data.Source.Epoch,
					Root:  data.Source.Root,
				},
				Target: &v1.Checkpoint{
					Epoch: data.Target.Epoch,
					Root:  data.Target.Root,
				},
			},
			InclusionDelay: att.InclusionDelay,
			ProposerIndex:  att.ProposerIndex,
		}
	}
	result := &v1.BeaconState{
		GenesisTime:           b.GenesisTime(),
		GenesisValidatorsRoot: b.GenesisValidatorRoot(),
		Slot:                  b.Slot(),
		Fork: &v1.Fork{
			PreviousVersion: sourceFork.PreviousVersion,
			CurrentVersion:  sourceFork.CurrentVersion,
			Epoch:           sourceFork.Epoch,
		},
		LatestBlockHeader: &v1.BeaconBlockHeader{
			Slot:          sourceLatestBlockHeader.Slot,
			ProposerIndex: sourceLatestBlockHeader.ProposerIndex,
			ParentRoot:    sourceLatestBlockHeader.ParentRoot,
			StateRoot:     sourceLatestBlockHeader.StateRoot,
			BodyRoot:      sourceLatestBlockHeader.BodyRoot,
		},
		BlockRoots:      b.BlockRoots(),
		StateRoots:      b.StateRoots(),
		HistoricalRoots: b.HistoricalRoots(),
		Eth1Data: &v1.Eth1Data{
			DepositRoot:  sourceEth1Data.DepositRoot,
			DepositCount: sourceEth1Data.DepositCount,
			BlockHash:    sourceEth1Data.BlockHash,
		},
		Eth1DataVotes:             resultEth1DataVotes,
		Eth1DepositIndex:          b.Eth1DepositIndex(),
		Validators:                resultValidators,
		Balances:                  b.Balances(),
		RandaoMixes:               b.RandaoMixes(),
		Slashings:                 b.Slashings(),
		PreviousEpochAttestations: resultPrevEpochAtts,
		CurrentEpochAttestations:  resultCurrEpochAtts,
		JustificationBits:         b.JustificationBits(),
		PreviousJustifiedCheckpoint: &v1.Checkpoint{
			Epoch: sourcePrevJustifiedCheckpoint.Epoch,
			Root:  sourcePrevJustifiedCheckpoint.Root,
		},
		CurrentJustifiedCheckpoint: &v1.Checkpoint{
			Epoch: sourceCurrJustifiedCheckpoint.Epoch,
			Root:  sourceCurrJustifiedCheckpoint.Root,
		},
		FinalizedCheckpoint: &v1.Checkpoint{
			Epoch: sourceFinalizedCheckpoint.Epoch,
			Root:  sourceFinalizedCheckpoint.Root,
		},
	}

	return result, nil
}

// FieldReferencesCount returns the reference count held by each field. This
// also includes the field trie held by each field.
func (b *BeaconState) FieldReferencesCount() map[string]uint64 {
	refMap := make(map[string]uint64)
	b.lock.RLock()
	defer b.lock.RUnlock()
	for i, f := range b.sharedFieldReferences {
		refMap[i.String()] = uint64(f.Refs())
	}
	for i, f := range b.stateFieldLeaves {
		numOfRefs := uint64(f.reference.Refs())
		f.RLock()
		if len(f.fieldLayers) != 0 {
			refMap[i.String()+"_trie"] = numOfRefs
		}
		f.RUnlock()
	}
	return refMap
}

// IsNil checks if the state and the underlying proto
// object are nil.
func (b *BeaconState) IsNil() bool {
	return b == nil || b.state == nil
}

func (b *BeaconState) rootSelector(ctx context.Context, field fieldIndex) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconState.rootSelector")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("field", field.String()))

	hasher := hashutil.CustomSHA256Hasher()
	switch field {
	case genesisTime:
		return htrutils.Uint64Root(b.state.GenesisTime), nil
	case genesisValidatorRoot:
		return bytesutil.ToBytes32(b.state.GenesisValidatorsRoot), nil
	case slot:
		return htrutils.Uint64Root(uint64(b.state.Slot)), nil
	case eth1DepositIndex:
		return htrutils.Uint64Root(b.state.Eth1DepositIndex), nil
	case fork:
		return htrutils.ForkRoot(b.state.Fork)
	case latestBlockHeader:
		return stateutil.BlockHeaderRoot(b.state.LatestBlockHeader)
	case blockRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.BlockRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(blockRoots, b.state.BlockRoots)
	case stateRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.StateRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(stateRoots, b.state.StateRoots)
	case historicalRoots:
		return htrutils.HistoricalRootsRoot(b.state.HistoricalRoots)
	case eth1Data:
		return eth1Root(hasher, b.state.Eth1Data)
	case eth1DataVotes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.state.Eth1DataVotes,
				uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))),
			)
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.state.Eth1DataVotes)
	case validators:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.Validators, params.BeaconConfig().ValidatorRegistryLimit)
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[validators] = []uint64{}
			delete(b.rebuildTrie, validators)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(validators, b.state.Validators)
	case balances:
		return stateutil.Uint64ListRootWithRegistryLimit(b.state.Balances)
	case randaoMixes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.RandaoMixes, uint64(params.BeaconConfig().EpochsPerHistoricalVector))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(randaoMixes, b.state.RandaoMixes)
	case slashings:
		return htrutils.SlashingsRoot(b.state.Slashings)
	case previousEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.state.PreviousEpochAttestations,
				uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)),
			)
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.state.PreviousEpochAttestations)
	case currentEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.state.CurrentEpochAttestations,
				uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)),
			)
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.state.CurrentEpochAttestations)
	case justificationBits:
		return bytesutil.ToBytes32(b.state.JustificationBits), nil
	case previousJustifiedCheckpoint:
		return htrutils.CheckpointRoot(hasher, b.state.PreviousJustifiedCheckpoint)
	case currentJustifiedCheckpoint:
		return htrutils.CheckpointRoot(hasher, b.state.CurrentJustifiedCheckpoint)
	case finalizedCheckpoint:
		return htrutils.CheckpointRoot(hasher, b.state.FinalizedCheckpoint)
	}
	return [32]byte{}, errors.New("invalid field index provided")
}

func (b *BeaconState) recomputeFieldTrie(index fieldIndex, elements interface{}) ([32]byte, error) {
	fTrie := b.stateFieldLeaves[index]
	if fTrie.reference.Refs() > 1 {
		fTrie.Lock()
		defer fTrie.Unlock()
		fTrie.reference.MinusRef()
		newTrie := fTrie.CopyTrie()
		b.stateFieldLeaves[index] = newTrie
		fTrie = newTrie
	}
	// remove duplicate indexes
	b.dirtyIndices[index] = sliceutil.SetUint64(b.dirtyIndices[index])
	// sort indexes again
	sort.Slice(b.dirtyIndices[index], func(i int, j int) bool {
		return b.dirtyIndices[index][i] < b.dirtyIndices[index][j]
	})
	root, err := fTrie.RecomputeTrie(b.dirtyIndices[index], elements)
	if err != nil {
		return [32]byte{}, err
	}
	b.dirtyIndices[index] = []uint64{}
	return root, nil
}

func (b *BeaconState) resetFieldTrie(index fieldIndex, elements interface{}, length uint64) error {
	fTrie, err := NewFieldTrie(index, elements, length)
	if err != nil {
		return err
	}
	b.stateFieldLeaves[index] = fTrie
	b.dirtyIndices[index] = []uint64{}
	return nil
}
