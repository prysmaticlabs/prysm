package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/minio/highwayhash"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func validatorBalancesRoot(balances []uint64) ([32]byte, error) {
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

func validatorRegistryRoot(validators []*ethpb.Validator) ([32]byte, error) {
	hashKeyElements := make([]byte, len(validators)*32)
	roots := make([][]byte, len(validators))
	emptyKey := highwayhash.Sum(hashKeyElements, fastSumHashKey[:])
	bytesProcessed := 0
	for i := 0; i < len(validators); i++ {
		val, err := validatorRoot(validators[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute validators merkleization")
		}
		copy(hashKeyElements[bytesProcessed:bytesProcessed+32], val[:])
		roots[i] = val[:]
		bytesProcessed += 32
	}

	hashKey := highwayhash.Sum(hashKeyElements, fastSumHashKey[:])
	if hashKey != emptyKey {
		if found, ok := rootsCache.Get(string(hashKey[:])); found != nil && ok {
			return found.([32]byte), nil
		}
	}

	validatorsRootsRoot, err := bitwiseMerkleize(roots, uint64(len(roots)), params.BeaconConfig().ValidatorRegistryLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	validatorsRootsBuf := new(bytes.Buffer)
	if err := binary.Write(validatorsRootsBuf, binary.LittleEndian, uint64(len(validators))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal validator registry length")
	}
	// We need to mix in the length of the slice.
	validatorsRootsBufRoot := make([]byte, 32)
	copy(validatorsRootsBufRoot, validatorsRootsBuf.Bytes())
	res := mixInLength(validatorsRootsRoot, validatorsRootsBufRoot)
	if hashKey != emptyKey {
		rootsCache.Set(string(hashKey[:]), res, 32)
	}
	return res, nil
}

func validatorRoot(validator *ethpb.Validator) ([32]byte, error) {
	// Validator marshaling for caching.
	enc := make([]byte, 122)
	copy(enc[0:48], validator.PublicKey)
	copy(enc[48:80], validator.WithdrawalCredentials)
	effectiveBalanceBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(effectiveBalanceBuf, validator.EffectiveBalance)
	copy(enc[80:88], effectiveBalanceBuf)
	if validator.Slashed {
		enc[88] = uint8(1)
	} else {
		enc[88] = uint8(0)
	}
	activationEligibilityBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(activationEligibilityBuf, validator.ActivationEligibilityEpoch)
	copy(enc[89:97], activationEligibilityBuf)

	activationBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(activationBuf, validator.ActivationEpoch)
	copy(enc[97:105], activationBuf)

	exitBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(exitBuf, validator.ExitEpoch)
	copy(enc[105:113], exitBuf)

	withdrawalBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(withdrawalBuf, validator.WithdrawableEpoch)
	copy(enc[113:121], exitBuf)

	// Check if it exists in cache:
	if found, ok := rootsCache.Get(string(enc)); found != nil && ok {
		return found.([32]byte), nil
	}

	fieldRoots := make([][]byte, 8)

	// Public key.
	pubKeyChunks, err := pack([][]byte{validator.PublicKey})
	if err != nil {
		return [32]byte{}, err
	}
	pubKeyRoot, err := bitwiseMerkleize(pubKeyChunks, uint64(len(pubKeyChunks)), uint64(len(pubKeyChunks)))
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots[0] = pubKeyRoot[:]

	// Withdrawal credentials.
	fieldRoots[1] = validator.WithdrawalCredentials

	// Effective balance.
	effBalRoot := bytesutil.ToBytes32(effectiveBalanceBuf)
	fieldRoots[2] = effBalRoot[:]

	// Slashed.
	slashBuf := make([]byte, 1)
	if validator.Slashed {
		slashBuf[0] = uint8(1)
	} else {
		slashBuf[0] = uint8(0)
	}
	slashBufRoot := bytesutil.ToBytes32(slashBuf)
	fieldRoots[3] = slashBufRoot[:]

	// Activation eligibility epoch.
	activationEligibilityRoot := bytesutil.ToBytes32(activationEligibilityBuf)
	fieldRoots[4] = activationEligibilityRoot[:]

	// Activation epoch.
	activationRoot := bytesutil.ToBytes32(activationBuf)
	fieldRoots[5] = activationRoot[:]

	// Exit epoch.
	exitBufRoot := bytesutil.ToBytes32(exitBuf)
	fieldRoots[6] = exitBufRoot[:]

	// Withdrawable epoch.
	withdrawalBufRoot := bytesutil.ToBytes32(withdrawalBuf)
	fieldRoots[7] = withdrawalBufRoot[:]

	valRoot, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, err
	}
	rootsCache.Set(string(enc), valRoot, 32)
	return valRoot, nil
}
