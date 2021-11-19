package stateutil

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ValidatorRootWithHasher describes a method from which the hash tree root
// of a validator is returned.
func ValidatorRootWithHasher(hasher ssz.HashFn, validator *ethpb.Validator) ([32]byte, error) {
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
		pubKeyChunks, err := ssz.Pack([][]byte{pubkey[:]})
		if err != nil {
			return [32]byte{}, err
		}
		pubKeyRoot, err := ssz.BitwiseMerkleize(hasher, pubKeyChunks, uint64(len(pubKeyChunks)), uint64(len(pubKeyChunks)))
		if err != nil {
			return [32]byte{}, err
		}
		fieldRoots = [][32]byte{pubKeyRoot, withdrawCreds, effectiveBalanceBuf, slashBuf, activationEligibilityBuf,
			activationBuf, exitBuf, withdrawalBuf}
	}
	return ssz.BitwiseMerkleizeArrays(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func merkleizeFlatArray(vec [][32]byte,
	depth uint8,
	hasher func([][32]byte, [][32]byte, uint64),
	zero_hash_array [][32]byte) [32]byte {

	if depth == 0 && len(vec) == 1 {
		return vec[0]
	}
	if len(vec) == 0 {
		panic("Can't have empty vec")
	}

	// allocate size for the buffer (everything hardcoded cause
	layer := (len(vec) + 1) / 2
	length := 0
	for {
		length += layer - 1
		if layer == 1 {
			break
		}
		layer = (layer + 1) / 2
	}
	length += int(depth)
	hash_tree := make([][32]byte, length)

	first := uint64(0)
	height := uint8(1)
	last := uint64(len(vec)+1) / 2
	if len(vec) > 1 {
		hasher(hash_tree, vec, last)
	}
	if len(vec)%2 == 1 {
		hash_tree[last-1] = hash.Hash2ChunksShani(vec[len(vec)-1], zero_hash_array[0])
	}
	for {
		dist := last - first
		if dist < 2 {
			break
		}
		hasher(hash_tree[last:], hash_tree[first:], dist/2)
		first = last
		last += (dist + 1) / 2

		if dist%2 != 0 {
			hash_tree[last-1] = hash.Hash2ChunksShani(hash_tree[first-1], zero_hash_array[height])
		}
		height++
	}
	for {
		if height >= depth {
			break
		}
		hash_tree[last] = hash.Hash2ChunksShani(hash_tree[last-1], zero_hash_array[height])
		last++
		height++
	}
	return hash_tree[last-1]
}

// Uint64ListRootWithRegistryLimitShani computes the HashTreeRoot Merkleization of
// a list of uint64 and mixed with registry limit. Flat array implementation
// using Shani extensions
func Uint64ListRootWithRegistryLimitShani(balances []uint64, zero_hash_array [][32]byte) ([32]byte, error) {
	// assume len(balances) is multiple of 4 for this benchmark
	lenChunks := len(balances) / 4
	balancesChunks := make([][32]byte, lenChunks)
	for i := 0; i < lenChunks; i++ {
		binary.LittleEndian.PutUint64(balancesChunks[i][:], balances[4*i])
		binary.LittleEndian.PutUint64(balancesChunks[i][8:], balances[4*i+1])
		binary.LittleEndian.PutUint64(balancesChunks[i][16:], balances[4*i+2])
		binary.LittleEndian.PutUint64(balancesChunks[i][24:], balances[4*i+3])
	}
	balancesRootsRoot := merkleizeFlatArray(balancesChunks, 38, hash.PotuzHasherShaniChunks, zero_hash_array)

	return hash.MixinLengthShani(balancesRootsRoot, uint64(len(balances))), nil
}

// Uint64ListRootWithRegistryLimit computes the HashTreeRoot Merkleization of
// a list of uint64 and mixed with registry limit.
func Uint64ListRootWithRegistryLimit(balances []uint64) ([32]byte, error) {
	hasher := hash.CustomSHA256Hasher()
	balancesMarshaling := make([][]byte, 0)
	for i := 0; i < len(balances); i++ {
		balanceBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(balanceBuf, balances[i])
		balancesMarshaling = append(balancesMarshaling, balanceBuf)
	}
	balancesChunks, err := ssz.Pack(balancesMarshaling)
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
	balancesRootsRoot, err := ssz.BitwiseMerkleize(hasher, balancesChunks, uint64(len(balancesChunks)), balLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute balances merkleization")
	}

	balancesLengthRoot := make([]byte, 32)
	binary.LittleEndian.PutUint64(balancesLengthRoot, uint64(len(balances)))
	return ssz.MixInLength(balancesRootsRoot, balancesLengthRoot), nil
}

// ValidatorEncKey returns the encoded key in bytes of input `validator`,
// the returned key bytes can be used for caching purposes.
func ValidatorEncKey(validator *ethpb.Validator) []byte {
	if validator == nil {
		return nil
	}

	enc := make([]byte, 122)
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

	return enc
}

// HandleValidatorSlice returns the validator indices in a slice of root format.
func HandleValidatorSlice(val []*ethpb.Validator, indices []uint64, convertAll bool) ([][32]byte, error) {
	length := len(indices)
	if convertAll {
		length = len(val)
	}
	roots := make([][32]byte, 0, length)
	hasher := hash.CustomSHA256Hasher()
	rootCreator := func(input *ethpb.Validator) error {
		newRoot, err := ValidatorRootWithHasher(hasher, input)
		if err != nil {
			return err
		}
		roots = append(roots, newRoot)
		return nil
	}
	if convertAll {
		for i := range val {
			err := rootCreator(val[i])
			if err != nil {
				return nil, err
			}
		}
		return roots, nil
	}
	if len(val) > 0 {
		for _, idx := range indices {
			if idx > uint64(len(val))-1 {
				return nil, fmt.Errorf("index %d greater than number of validators %d", idx, len(val))
			}
			err := rootCreator(val[idx])
			if err != nil {
				return nil, err
			}
		}
	}
	return roots, nil
}
