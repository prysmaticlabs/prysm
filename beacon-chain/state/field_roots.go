package state

import (
	"bytes"
	"encoding/binary"
	"sync"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	leavesCache = make(map[string][][32]byte, params.BeaconConfig().BeaconStateFieldCount)
	layersCache = make(map[string][][][32]byte, params.BeaconConfig().BeaconStateFieldCount)
	lock        sync.RWMutex
)

const cacheSize = 100000

var nocachedHasher *stateRootHasher
var cachedHasher *stateRootHasher

func init() {
	rootsCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: cacheSize, // number of keys to track frequency of (1M).
		MaxCost:     1 << 22,   // maximum cost of cache (3MB).
		// 100,000 roots will take up approximately 3 MB in memory.
		BufferItems: 64, // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}
	// Temporarily disable roots cache until cache issues can be resolved.
	cachedHasher = &stateRootHasher{rootsCache: rootsCache}
	nocachedHasher = &stateRootHasher{}
}

type stateRootHasher struct {
	rootsCache *ristretto.Cache
}

// computeFieldRoots returns the hash tree root computations of every field in
// the beacon state as a list of 32 byte roots.
func computeFieldRoots(state *pb.BeaconState) ([][]byte, error) {
	if featureconfig.Get().EnableSSZCache {
		return cachedHasher.computeFieldRootsWithHasher(state)
	}
	return nocachedHasher.computeFieldRootsWithHasher(state)
}

func (h *stateRootHasher) computeFieldRootsWithHasher(state *pb.BeaconState) ([][]byte, error) {
	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hashutil.CustomSHA256Hasher()
	fieldRoots := make([][]byte, params.BeaconConfig().BeaconStateFieldCount)

	// Genesis time root.
	genesisRoot := htrutils.Uint64Root(state.GenesisTime)
	fieldRoots[0] = genesisRoot[:]

	// Genesis validator root.
	r := [32]byte{}
	copy(r[:], state.GenesisValidatorsRoot)
	fieldRoots[1] = r[:]

	// Slot root.
	slotRoot := htrutils.Uint64Root(uint64(state.Slot))
	fieldRoots[2] = slotRoot[:]

	// Fork data structure root.
	forkHashTreeRoot, err := htrutils.ForkRoot(state.Fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[3] = forkHashTreeRoot[:]

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := blockHeaderRoot(state.LatestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[4] = headerHashTreeRoot[:]

	// BlockRoots array root.
	blockRootsRoot, err := h.arraysRoot(state.BlockRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot), "BlockRoots")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[5] = blockRootsRoot[:]

	// StateRoots array root.
	stateRootsRoot, err := h.arraysRoot(state.StateRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot), "StateRoots")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[6] = stateRootsRoot[:]

	// HistoricalRoots slice root.
	historicalRootsRt, err := htrutils.HistoricalRootsRoot(state.HistoricalRoots)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[7] = historicalRootsRt[:]

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := eth1Root(hasher, state.Eth1Data)
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
	validatorsRoot, err := h.validatorRegistryRoot(state.Validators)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[11] = validatorsRoot[:]

	// Balances slice root.
	balancesRoot, err := validatorBalancesRoot(state.Balances)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[12] = balancesRoot[:]

	// RandaoMixes array root.
	randaoRootsRoot, err := h.arraysRoot(state.RandaoMixes, uint64(params.BeaconConfig().EpochsPerHistoricalVector), "RandaoMixes")
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[13] = randaoRootsRoot[:]

	// Slashings array root.
	slashingsRootsRoot, err := htrutils.SlashingsRoot(state.Slashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[14] = slashingsRootsRoot[:]

	// PreviousEpochAttestations slice root.
	prevAttsRoot, err := h.epochAttestationsRoot(state.PreviousEpochAttestations)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
	}
	fieldRoots[15] = prevAttsRoot[:]

	// CurrentEpochAttestations slice root.
	currAttsRoot, err := h.epochAttestationsRoot(state.CurrentEpochAttestations)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current epoch attestations merkleization")
	}
	fieldRoots[16] = currAttsRoot[:]

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.JustificationBits)
	fieldRoots[17] = justifiedBitsRoot[:]

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := htrutils.CheckpointRoot(hasher, state.PreviousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[18] = prevCheckRoot[:]

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := htrutils.CheckpointRoot(hasher, state.CurrentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[19] = currJustRoot[:]

	// FinalizedCheckpoint data structure root.
	finalRoot, err := htrutils.CheckpointRoot(hasher, state.FinalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[20] = finalRoot[:]
	return fieldRoots, nil
}

// blockHeaderRoot computes the HashTreeRoot Merkleization of
// a BeaconBlockHeader struct according to the eth2
// Simple Serialize specification.
func blockHeaderRoot(header *ethpb.BeaconBlockHeader) ([32]byte, error) {
	fieldRoots := make([][]byte, 5)
	if header != nil {
		headerSlotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(headerSlotBuf, uint64(header.Slot))
		headerSlotRoot := bytesutil.ToBytes32(headerSlotBuf)
		fieldRoots[0] = headerSlotRoot[:]
		proposerIdxBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(proposerIdxBuf, uint64(header.ProposerIndex))
		proposerIndexRoot := bytesutil.ToBytes32(proposerIdxBuf)
		fieldRoots[1] = proposerIndexRoot[:]
		parentRoot := bytesutil.ToBytes32(header.ParentRoot)
		fieldRoots[2] = parentRoot[:]
		stateRoot := bytesutil.ToBytes32(header.StateRoot)
		fieldRoots[3] = stateRoot[:]
		bodyRoot := bytesutil.ToBytes32(header.BodyRoot)
		fieldRoots[4] = bodyRoot[:]
	}
	return htrutils.BitwiseMerkleize(hashutil.CustomSHA256Hasher(), fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// eth1Root computes the HashTreeRoot Merkleization of
// a BeaconBlockHeader struct according to the eth2
// Simple Serialize specification.
func eth1Root(hasher htrutils.HashFn, eth1Data *ethpb.Eth1Data) ([32]byte, error) {
	enc := make([]byte, 0, 96)
	fieldRoots := make([][]byte, 3)
	for i := 0; i < len(fieldRoots); i++ {
		fieldRoots[i] = make([]byte, 32)
	}
	if eth1Data != nil {
		if len(eth1Data.DepositRoot) > 0 {
			depRoot := bytesutil.ToBytes32(eth1Data.DepositRoot)
			fieldRoots[0] = depRoot[:]
			enc = append(enc, depRoot[:]...)
		}
		eth1DataCountBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(eth1DataCountBuf, eth1Data.DepositCount)
		eth1CountRoot := bytesutil.ToBytes32(eth1DataCountBuf)
		fieldRoots[1] = eth1CountRoot[:]
		enc = append(enc, eth1CountRoot[:]...)
		if len(eth1Data.BlockHash) > 0 {
			blockHash := bytesutil.ToBytes32(eth1Data.BlockHash)
			fieldRoots[2] = blockHash[:]
			enc = append(enc, blockHash[:]...)
		}
		if featureconfig.Get().EnableSSZCache {
			if found, ok := cachedHasher.rootsCache.Get(string(enc)); ok && found != nil {
				return found.([32]byte), nil
			}
		}
	}
	root, err := htrutils.BitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	if featureconfig.Get().EnableSSZCache {
		cachedHasher.rootsCache.Set(string(enc), root, 32)
	}
	return root, nil
}

// eth1DataVotesRoot computes the HashTreeRoot Merkleization of
// a list of Eth1Data structs according to the eth2
// Simple Serialize specification.
func eth1DataVotesRoot(eth1DataVotes []*ethpb.Eth1Data) ([32]byte, error) {
	eth1VotesRoots := make([][]byte, 0)
	enc := make([]byte, len(eth1DataVotes)*32)
	hasher := hashutil.CustomSHA256Hasher()
	for i := 0; i < len(eth1DataVotes); i++ {
		eth1, err := eth1Root(hasher, eth1DataVotes[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute eth1data merkleization")
		}
		copy(enc[(i*32):(i+1)*32], eth1[:])
		eth1VotesRoots = append(eth1VotesRoots, eth1[:])
	}
	hashKey := hashutil.FastSum256(enc)
	if featureconfig.Get().EnableSSZCache {
		if found, ok := cachedHasher.rootsCache.Get(string(hashKey[:])); ok && found != nil {
			return found.([32]byte), nil
		}
	}
	eth1Chunks, err := htrutils.Pack(eth1VotesRoots)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not chunk eth1 votes roots")
	}
	eth1VotesRootsRoot, err := htrutils.BitwiseMerkleize(
		hasher,
		eth1Chunks,
		uint64(len(eth1Chunks)),
		uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))),
	)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	eth1VotesRootBuf := new(bytes.Buffer)
	if err := binary.Write(eth1VotesRootBuf, binary.LittleEndian, uint64(len(eth1DataVotes))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal eth1data votes length")
	}
	// We need to mix in the length of the slice.
	eth1VotesRootBufRoot := make([]byte, 32)
	copy(eth1VotesRootBufRoot, eth1VotesRootBuf.Bytes())
	root := htrutils.MixInLength(eth1VotesRootsRoot, eth1VotesRootBufRoot)
	if featureconfig.Get().EnableSSZCache {
		cachedHasher.rootsCache.Set(string(hashKey[:]), root, 32)
	}
	return root, nil
}

func (h *stateRootHasher) validatorRegistryRoot(validators []*ethpb.Validator) ([32]byte, error) {
	hashKeyElements := make([]byte, len(validators)*32)
	roots := make([][32]byte, len(validators))
	emptyKey := hashutil.FastSum256(hashKeyElements)
	hasher := hashutil.CustomSHA256Hasher()
	bytesProcessed := 0
	for i := 0; i < len(validators); i++ {
		val, err := h.validatorRoot(hasher, validators[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute validators merkleization")
		}
		copy(hashKeyElements[bytesProcessed:bytesProcessed+32], val[:])
		roots[i] = val
		bytesProcessed += 32
	}

	hashKey := hashutil.FastSum256(hashKeyElements)
	if hashKey != emptyKey && h.rootsCache != nil {
		if found, ok := h.rootsCache.Get(string(hashKey[:])); found != nil && ok {
			return found.([32]byte), nil
		}
	}

	validatorsRootsRoot, err := htrutils.BitwiseMerkleizeArrays(hasher, roots, uint64(len(roots)), params.BeaconConfig().ValidatorRegistryLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	validatorsRootsBuf := new(bytes.Buffer)
	if err := binary.Write(validatorsRootsBuf, binary.LittleEndian, uint64(len(validators))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal validator registry length")
	}
	// We need to mix in the length of the slice.
	var validatorsRootsBufRoot [32]byte
	copy(validatorsRootsBufRoot[:], validatorsRootsBuf.Bytes())
	res := htrutils.MixInLength(validatorsRootsRoot, validatorsRootsBufRoot[:])
	if hashKey != emptyKey && h.rootsCache != nil {
		h.rootsCache.Set(string(hashKey[:]), res, 32)
	}
	return res, nil
}

func (h *stateRootHasher) validatorRoot(hasher htrutils.HashFn, validator *ethpb.Validator) ([32]byte, error) {
	// Validator marshaling for caching.
	enc := make([]byte, 122)
	fieldRoots := make([][32]byte, 2, 8)

	if validator != nil {
		pubkey := bytesutil.ToBytes48(validator.PublicKey)
		copy(enc[0:48], pubkey[:])
		withdrawCreds := bytesutil.ToBytes32(validator.WithdrawalCredentials)
		copy(enc[48:80], withdrawCreds[:])
		effectiveBalanceBuf := [32]byte{}
		binary.LittleEndian.PutUint64(effectiveBalanceBuf[:8], validator.EffectiveBalance)
		copy(enc[80:88], effectiveBalanceBuf[:8])
		if validator.Slashed {
			enc[88] = uint8(1)
		} else {
			enc[88] = uint8(0)
		}
		activationEligibilityBuf := [32]byte{}
		binary.LittleEndian.PutUint64(activationEligibilityBuf[:8], uint64(validator.ActivationEligibilityEpoch))
		copy(enc[89:97], activationEligibilityBuf[:8])

		activationBuf := [32]byte{}
		binary.LittleEndian.PutUint64(activationBuf[:8], uint64(validator.ActivationEpoch))
		copy(enc[97:105], activationBuf[:8])

		exitBuf := [32]byte{}
		binary.LittleEndian.PutUint64(exitBuf[:8], uint64(validator.ExitEpoch))
		copy(enc[105:113], exitBuf[:8])

		withdrawalBuf := [32]byte{}
		binary.LittleEndian.PutUint64(withdrawalBuf[:8], uint64(validator.WithdrawableEpoch))
		copy(enc[113:121], withdrawalBuf[:8])

		// Check if it exists in cache:
		if h.rootsCache != nil {
			if found, ok := h.rootsCache.Get(string(enc)); found != nil && ok {
				return found.([32]byte), nil
			}
		}

		// Public key.
		pubKeyChunks, err := htrutils.Pack([][]byte{pubkey[:]})
		if err != nil {
			return [32]byte{}, err
		}
		pubKeyRoot, err := htrutils.BitwiseMerkleize(hasher, pubKeyChunks, uint64(len(pubKeyChunks)), uint64(len(pubKeyChunks)))
		if err != nil {
			return [32]byte{}, err
		}
		fieldRoots[0] = pubKeyRoot

		// Withdrawal credentials.
		copy(fieldRoots[1][:], withdrawCreds[:])

		// Effective balance.
		fieldRoots = append(fieldRoots, effectiveBalanceBuf)

		// Slashed.
		slashBuf := [32]byte{}
		if validator.Slashed {
			slashBuf[0] = uint8(1)
		} else {
			slashBuf[0] = uint8(0)
		}
		fieldRoots = append(
			fieldRoots,
			slashBuf,
			activationEligibilityBuf,
			activationBuf,
			exitBuf,
			withdrawalBuf,
		)
	}

	valRoot, err := htrutils.BitwiseMerkleizeArrays(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	if h.rootsCache != nil {
		h.rootsCache.Set(string(enc), valRoot, 32)
	}
	return valRoot, nil
}

// validatorBalancesRoot computes the HashTreeRoot Merkleization of
// a list of validator uint64 balances according to the eth2
// Simple Serialize specification.
func validatorBalancesRoot(balances []uint64) ([32]byte, error) {
	hasher := hashutil.CustomSHA256Hasher()
	balancesMarshaling := make([][]byte, 0)
	for i := 0; i < len(balances); i++ {
		balanceBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(balanceBuf, balances[i])
		balancesMarshaling = append(balancesMarshaling, balanceBuf)
	}
	balancesChunks, err := htrutils.Pack(balancesMarshaling)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not pack balances into chunks")
	}
	maxBalCap := params.BeaconConfig().ValidatorRegistryLimit
	elemSize := uint64(8)
	balLimit := (maxBalCap*elemSize + 31) / 32
	if balLimit == 0 {
		if len(balances) == 0 {
			balLimit = 1
		} else {
			balLimit = uint64(len(balances))
		}
	}
	balancesRootsRoot, err := htrutils.BitwiseMerkleize(hasher, balancesChunks, uint64(len(balancesChunks)), balLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute balances merkleization")
	}
	balancesRootsBuf := new(bytes.Buffer)
	if err := binary.Write(balancesRootsBuf, binary.LittleEndian, uint64(len(balances))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal balances length")
	}
	balancesRootsBufRoot := make([]byte, 32)
	copy(balancesRootsBufRoot, balancesRootsBuf.Bytes())
	return htrutils.MixInLength(balancesRootsRoot, balancesRootsBufRoot), nil
}

// ValidatorRegistryRoot computes the HashTreeRoot Merkleization of
// a list of validator structs according to the eth2
// Simple Serialize specification.
func ValidatorRegistryRoot(vals []*ethpb.Validator) ([32]byte, error) {
	if featureconfig.Get().EnableSSZCache {
		return cachedHasher.validatorRegistryRoot(vals)
	}
	return nocachedHasher.validatorRegistryRoot(vals)
}

// ValidatorRoot describes a method from which the hash tree root
// of a validator is returned.
func ValidatorRoot(hasher htrutils.HashFn, validator *ethpb.Validator) ([32]byte, error) {
	var fieldRoots [][32]byte
	if validator != nil {
		pubkey := bytesutil.ToBytes48(validator.PublicKey)
		withdrawCreds := bytesutil.ToBytes32(validator.WithdrawalCredentials)
		effectiveBalanceBuf := [32]byte{}
		binary.LittleEndian.PutUint64(effectiveBalanceBuf[:8], validator.EffectiveBalance)
		// Slashed.
		slashBuf := [32]byte{}
		if validator.Slashed {
			slashBuf[0] = uint8(1)
		} else {
			slashBuf[0] = uint8(0)
		}
		activationEligibilityBuf := [32]byte{}
		binary.LittleEndian.PutUint64(activationEligibilityBuf[:8], uint64(validator.ActivationEligibilityEpoch))

		activationBuf := [32]byte{}
		binary.LittleEndian.PutUint64(activationBuf[:8], uint64(validator.ActivationEpoch))

		exitBuf := [32]byte{}
		binary.LittleEndian.PutUint64(exitBuf[:8], uint64(validator.ExitEpoch))

		withdrawalBuf := [32]byte{}
		binary.LittleEndian.PutUint64(withdrawalBuf[:8], uint64(validator.WithdrawableEpoch))

		// Public key.
		pubKeyChunks, err := htrutils.Pack([][]byte{pubkey[:]})
		if err != nil {
			return [32]byte{}, err
		}
		pubKeyRoot, err := htrutils.BitwiseMerkleize(hasher, pubKeyChunks, uint64(len(pubKeyChunks)), uint64(len(pubKeyChunks)))
		if err != nil {
			return [32]byte{}, err
		}
		fieldRoots = [][32]byte{pubKeyRoot, withdrawCreds, effectiveBalanceBuf, slashBuf, activationEligibilityBuf,
			activationBuf, exitBuf, withdrawalBuf}
	}
	return htrutils.BitwiseMerkleizeArrays(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func (h *stateRootHasher) epochAttestationsRoot(atts []*pb.PendingAttestation) ([32]byte, error) {
	hasher := hashutil.CustomSHA256Hasher()
	roots := make([][]byte, len(atts))
	for i := 0; i < len(atts); i++ {
		pendingRoot, err := h.pendingAttestationRoot(hasher, atts[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not attestation merkleization")
		}
		roots[i] = pendingRoot[:]
	}

	attsRootsRoot, err := htrutils.BitwiseMerkleize(
		hasher,
		roots,
		uint64(len(roots)),
		uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)),
	)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute epoch attestations merkleization")
	}
	attsLenBuf := new(bytes.Buffer)
	if err := binary.Write(attsLenBuf, binary.LittleEndian, uint64(len(atts))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal epoch attestations length")
	}
	// We need to mix in the length of the slice.
	attsLenRoot := make([]byte, 32)
	copy(attsLenRoot, attsLenBuf.Bytes())
	res := htrutils.MixInLength(attsRootsRoot, attsLenRoot)
	return res, nil
}

func (h *stateRootHasher) pendingAttestationRoot(hasher htrutils.HashFn, att *pb.PendingAttestation) ([32]byte, error) {
	// Marshal attestation to determine if it exists in the cache.
	enc := make([]byte, 2192)
	fieldRoots := make([][]byte, 4)

	if att != nil {
		copy(enc[0:2048], att.AggregationBits)

		inclusionBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(inclusionBuf, uint64(att.InclusionDelay))
		copy(enc[2048:2056], inclusionBuf)

		attDataBuf := marshalAttestationData(att.Data)
		copy(enc[2056:2184], attDataBuf)

		proposerBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(proposerBuf, uint64(att.ProposerIndex))
		copy(enc[2184:2192], proposerBuf)

		// Check if it exists in cache:
		if h.rootsCache != nil {
			if found, ok := h.rootsCache.Get(string(enc)); found != nil && ok {
				return found.([32]byte), nil
			}
		}

		// Bitfield.
		aggregationRoot, err := htrutils.BitlistRoot(hasher, att.AggregationBits, 2048)
		if err != nil {
			return [32]byte{}, err
		}
		fieldRoots[0] = aggregationRoot[:]

		// Attestation data.
		attDataRoot, err := attestationDataRoot(hasher, att.Data)
		if err != nil {
			return [32]byte{}, err
		}
		fieldRoots[1] = attDataRoot[:]

		// Inclusion delay.
		inclusionRoot := bytesutil.ToBytes32(inclusionBuf)
		fieldRoots[2] = inclusionRoot[:]

		// Proposer index.
		proposerRoot := bytesutil.ToBytes32(proposerBuf)
		fieldRoots[3] = proposerRoot[:]
	}
	res, err := htrutils.BitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	if h.rootsCache != nil {
		h.rootsCache.Set(string(enc), res, 32)
	}
	return res, nil
}

func marshalAttestationData(data *ethpb.AttestationData) []byte {
	enc := make([]byte, 128)

	if data != nil {
		// Slot.
		slotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(slotBuf, uint64(data.Slot))
		copy(enc[0:8], slotBuf)

		// Committee index.
		indexBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(indexBuf, uint64(data.CommitteeIndex))
		copy(enc[8:16], indexBuf)

		copy(enc[16:48], data.BeaconBlockRoot)

		// Source epoch and root.
		if data.Source != nil {
			sourceEpochBuf := make([]byte, 8)
			binary.LittleEndian.PutUint64(sourceEpochBuf, uint64(data.Source.Epoch))
			copy(enc[48:56], sourceEpochBuf)
			copy(enc[56:88], data.Source.Root)
		}

		// Target.
		if data.Target != nil {
			targetEpochBuf := make([]byte, 8)
			binary.LittleEndian.PutUint64(targetEpochBuf, uint64(data.Target.Epoch))
			copy(enc[88:96], targetEpochBuf)
			copy(enc[96:128], data.Target.Root)
		}
	}

	return enc
}

func attestationDataRoot(hasher htrutils.HashFn, data *ethpb.AttestationData) ([32]byte, error) {
	fieldRoots := make([][]byte, 5)

	if data != nil {
		// Slot.
		slotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(slotBuf, uint64(data.Slot))
		slotRoot := bytesutil.ToBytes32(slotBuf)
		fieldRoots[0] = slotRoot[:]

		// CommitteeIndex.
		indexBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(indexBuf, uint64(data.CommitteeIndex))
		interRoot := bytesutil.ToBytes32(indexBuf)
		fieldRoots[1] = interRoot[:]

		// Beacon block root.
		blockRoot := bytesutil.ToBytes32(data.BeaconBlockRoot)
		fieldRoots[2] = blockRoot[:]

		// Source
		sourceRoot, err := htrutils.CheckpointRoot(hasher, data.Source)
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute source checkpoint merkleization")
		}
		fieldRoots[3] = sourceRoot[:]

		// Target
		targetRoot, err := htrutils.CheckpointRoot(hasher, data.Target)
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute target checkpoint merkleization")
		}
		fieldRoots[4] = targetRoot[:]
	}

	return htrutils.BitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func (h *stateRootHasher) arraysRoot(input [][]byte, length uint64, fieldName string) ([32]byte, error) {
	lock.Lock()
	defer lock.Unlock()
	hashFunc := hashutil.CustomSHA256Hasher()
	if _, ok := layersCache[fieldName]; !ok && h.rootsCache != nil {
		depth := htrutils.Depth(length)
		layersCache[fieldName] = make([][][32]byte, depth+1)
	}

	leaves := make([][32]byte, length)
	for i, chunk := range input {
		copy(leaves[i][:], chunk)
	}
	bytesProcessed := 0
	changedIndices := make([]int, 0)
	prevLeaves, ok := leavesCache[fieldName]
	if len(prevLeaves) == 0 || h.rootsCache == nil {
		prevLeaves = leaves
	}

	for i := 0; i < len(leaves); i++ {
		// We check if any items changed since the roots were last recomputed.
		notEqual := leaves[i] != prevLeaves[i]
		if ok && h.rootsCache != nil && notEqual {
			changedIndices = append(changedIndices, i)
		}
		bytesProcessed += 32
	}
	if len(changedIndices) > 0 && h.rootsCache != nil {
		var rt [32]byte
		var err error
		// If indices did change since last computation, we only recompute
		// the modified branches in the cached Merkle tree for this state field.
		chunks := leaves

		// We need to ensure we recompute indices of the Merkle tree which
		// changed in-between calls to this function. This check adds an offset
		// to the recomputed indices to ensure we do so evenly.
		maxChangedIndex := changedIndices[len(changedIndices)-1]
		if maxChangedIndex+2 == len(chunks) && maxChangedIndex%2 != 0 {
			changedIndices = append(changedIndices, maxChangedIndex+1)
		}
		for i := 0; i < len(changedIndices); i++ {
			rt, err = recomputeRoot(changedIndices[i], chunks, fieldName, hashFunc)
			if err != nil {
				return [32]byte{}, err
			}
		}
		leavesCache[fieldName] = chunks
		return rt, nil
	}

	res := h.merkleizeWithCache(leaves, length, fieldName, hashFunc)
	if h.rootsCache != nil {
		leavesCache[fieldName] = leaves
	}
	return res, nil
}

func recomputeRoot(idx int, chunks [][32]byte, fieldName string, hasher func([]byte) [32]byte) ([32]byte, error) {
	items, ok := layersCache[fieldName]
	if !ok {
		return [32]byte{}, errors.New("could not recompute root as there was no cache found")
	}
	if items == nil {
		return [32]byte{}, errors.New("could not recompute root as there were no items found in the layers cache")
	}
	layers := items
	root := chunks[idx]
	layers[0] = chunks
	// The merkle tree structure looks as follows:
	// [[r1, r2, r3, r4], [parent1, parent2], [root]]
	// Using information about the index which changed, idx, we recompute
	// only its branch up the tree.
	currentIndex := idx
	for i := 0; i < len(layers)-1; i++ {
		isLeft := currentIndex%2 == 0
		neighborIdx := currentIndex ^ 1

		neighbor := [32]byte{}
		if layers[i] != nil && len(layers[i]) != 0 && neighborIdx < len(layers[i]) {
			neighbor = layers[i][neighborIdx]
		}
		if isLeft {
			parentHash := hasher(append(root[:], neighbor[:]...))
			root = parentHash
		} else {
			parentHash := hasher(append(neighbor[:], root[:]...))
			root = parentHash
		}
		parentIdx := currentIndex / 2
		// Update the cached layers at the parent index.
		if len(layers[i+1]) == 0 {
			layers[i+1] = append(layers[i+1], root)
		} else {
			layers[i+1][parentIdx] = root
		}
		currentIndex = parentIdx
	}
	layersCache[fieldName] = layers
	// If there is only a single leaf, we return it (the identity element).
	if len(layers[0]) == 1 {
		return layers[0][0], nil
	}
	return root, nil
}

func (h *stateRootHasher) merkleizeWithCache(leaves [][32]byte, length uint64,
	fieldName string, hasher func([]byte) [32]byte) [32]byte {
	if len(leaves) == 1 {
		return leaves[0]
	}
	hashLayer := leaves
	layers := make([][][32]byte, htrutils.Depth(length)+1)
	if items, ok := layersCache[fieldName]; ok && h.rootsCache != nil {
		if len(items[0]) == len(leaves) {
			layers = items
		}
	}
	layers[0] = hashLayer
	layers, hashLayer = merkleizeTrieLeaves(layers, hashLayer, hasher)
	root := hashLayer[0]
	if h.rootsCache != nil {
		layersCache[fieldName] = layers
	}
	return root
}

func merkleizeTrieLeaves(layers [][][32]byte, hashLayer [][32]byte,
	hasher func([]byte) [32]byte) ([][][32]byte, [][32]byte) {
	// We keep track of the hash layers of a Merkle trie until we reach
	// the top layer of length 1, which contains the single root element.
	//        [Root]      -> Top layer has length 1.
	//    [E]       [F]   -> This layer has length 2.
	// [A]  [B]  [C]  [D] -> The bottom layer has length 4 (needs to be a power of two).
	i := 1
	chunkBuffer := bytes.NewBuffer([]byte{})
	chunkBuffer.Grow(64)
	for len(hashLayer) > 1 && i < len(layers) {
		layer := make([][32]byte, len(hashLayer)/2)
		for j := 0; j < len(hashLayer); j += 2 {
			chunkBuffer.Write(hashLayer[j][:])
			chunkBuffer.Write(hashLayer[j+1][:])
			hashedChunk := hasher(chunkBuffer.Bytes())
			layer[j/2] = hashedChunk
			chunkBuffer.Reset()
		}
		hashLayer = layer
		layers[i] = hashLayer
		i++
	}
	return layers, hashLayer
}

// PendingAttestationRoot describes a method from which the hash tree root
// of a pending attestation is returned.
func PendingAttestationRoot(hasher htrutils.HashFn, att *pb.PendingAttestation) ([32]byte, error) {
	var fieldRoots [][32]byte
	if att != nil {
		// Bitfield.
		aggregationRoot, err := htrutils.BitlistRoot(hasher, att.AggregationBits, params.BeaconConfig().MaxValidatorsPerCommittee)
		if err != nil {
			return [32]byte{}, err
		}
		// Attestation data.
		attDataRoot, err := attestationDataRoot(hasher, att.Data)
		if err != nil {
			return [32]byte{}, err
		}
		inclusionBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(inclusionBuf, uint64(att.InclusionDelay))
		// Inclusion delay.
		inclusionRoot := bytesutil.ToBytes32(inclusionBuf)

		proposerBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(proposerBuf, uint64(att.ProposerIndex))
		// Proposer index.
		proposerRoot := bytesutil.ToBytes32(proposerBuf)

		fieldRoots = [][32]byte{aggregationRoot, attDataRoot, inclusionRoot, proposerRoot}
	}
	return htrutils.BitwiseMerkleizeArrays(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}
