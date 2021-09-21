package stateutil

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	butil "github.com/prysmaticlabs/prysm/encoding/bytes"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ValidatorRootWithHasher describes a method from which the hash tree root
// of a validator is returned.
func ValidatorRootWithHasher(hasher ssz.HashFn, validator *ethpb.Validator) ([32]byte, error) {
	var fieldRoots [][32]byte
	if validator != nil {
		pubkey := butil.ToBytes48(validator.PublicKey)
		withdrawCreds := butil.ToBytes32(validator.WithdrawalCredentials)
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
	pubkey := butil.ToBytes48(validator.PublicKey)
	copy(enc[0:48], pubkey[:])
	withdrawCreds := butil.ToBytes32(validator.WithdrawalCredentials)
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
