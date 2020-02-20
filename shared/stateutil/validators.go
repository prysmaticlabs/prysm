package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ValidatorRegistryRoot computes the HashTreeRoot Merkleization of
// a list of validator structs according to the eth2
// Simple Serialize specification.
func ValidatorRegistryRoot(vals []*ethpb.Validator) ([32]byte, error) {
	if featureconfig.Get().EnableSSZCache {
		return cachedHasher.validatorRegistryRoot(vals)
	}
	return nocachedHasher.validatorRegistryRoot(vals)
}

// ValidatorBalancesRoot computes the HashTreeRoot Merkleization of
// a list of validator uint64 balances according to the eth2
// Simple Serialize specification.
func ValidatorBalancesRoot(balances []uint64) ([32]byte, error) {
	balancesMarshaling := make([][]byte, 0)
	for i := 0; i < len(balances); i++ {
		balanceBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(balanceBuf, balances[i])
		balancesMarshaling = append(balancesMarshaling, balanceBuf)
	}
	balancesChunks, err := pack(balancesMarshaling)
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
	balancesRootsRoot, err := bitwiseMerkleize(balancesChunks, uint64(len(balancesChunks)), balLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute balances merkleization")
	}
	balancesRootsBuf := new(bytes.Buffer)
	if err := binary.Write(balancesRootsBuf, binary.LittleEndian, uint64(len(balances))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal balances length")
	}
	balancesRootsBufRoot := make([]byte, 32)
	copy(balancesRootsBufRoot, balancesRootsBuf.Bytes())
	return mixInLength(balancesRootsRoot, balancesRootsBufRoot), nil
}

func (h *stateRootHasher) validatorRegistryRoot(validators []*ethpb.Validator) ([32]byte, error) {
	hashKeyElements := make([]byte, len(validators)*32)
	roots := make([][32]byte, len(validators))
	emptyKey := hashutil.FastSum256(hashKeyElements)
	bytesProcessed := 0
	for i := 0; i < len(validators); i++ {
		val, err := h.validatorRoot(validators[i])
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

	validatorsRootsRoot, err := bitwiseMerkleizeArrays(roots, uint64(len(roots)), params.BeaconConfig().ValidatorRegistryLimit)
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
	res := mixInLength(validatorsRootsRoot, validatorsRootsBufRoot[:])
	if hashKey != emptyKey && h.rootsCache != nil {
		h.rootsCache.Set(string(hashKey[:]), res, 32)
	}
	return res, nil
}

func (h *stateRootHasher) validatorRoot(validator *ethpb.Validator) ([32]byte, error) {
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
		binary.LittleEndian.PutUint64(activationEligibilityBuf[:8], validator.ActivationEligibilityEpoch)
		copy(enc[89:97], activationEligibilityBuf[:8])

		activationBuf := [32]byte{}
		binary.LittleEndian.PutUint64(activationBuf[:8], validator.ActivationEpoch)
		copy(enc[97:105], activationBuf[:8])

		exitBuf := [32]byte{}
		binary.LittleEndian.PutUint64(exitBuf[:8], validator.ExitEpoch)
		copy(enc[105:113], exitBuf[:8])

		withdrawalBuf := [32]byte{}
		binary.LittleEndian.PutUint64(withdrawalBuf[:8], validator.WithdrawableEpoch)
		copy(enc[113:121], withdrawalBuf[:8])

		// Check if it exists in cache:
		if h.rootsCache != nil {
			if found, ok := h.rootsCache.Get(string(enc)); found != nil && ok {
				return found.([32]byte), nil
			}
		}

		// Public key.
		pubKeyChunks, err := pack([][]byte{pubkey[:]})
		if err != nil {
			return [32]byte{}, err
		}
		pubKeyRoot, err := bitwiseMerkleize(pubKeyChunks, uint64(len(pubKeyChunks)), uint64(len(pubKeyChunks)))
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
		fieldRoots = append(fieldRoots, slashBuf)

		// Activation eligibility epoch.
		fieldRoots = append(fieldRoots, activationEligibilityBuf)

		// Activation epoch.
		fieldRoots = append(fieldRoots, activationBuf)

		// Exit epoch.
		fieldRoots = append(fieldRoots, exitBuf)

		// Withdrawable epoch.
		fieldRoots = append(fieldRoots, withdrawalBuf)
	}

	valRoot, err := bitwiseMerkleizeArrays(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	if h.rootsCache != nil {
		h.rootsCache.Set(string(enc), valRoot, 32)
	}
	return valRoot, nil
}
