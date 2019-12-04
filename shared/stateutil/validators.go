package stateutil

import (
	"encoding/binary"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func validatorRoot(validator *ethpb.Validator) ([32]byte, error) {
	fieldRoots := make([][]byte, 8)

	// Public key.
	pubKeyChunks, err := pack([][]byte{validator.PublicKey})
	if err != nil {
		return [32]byte{}, nil
	}
	pubKeyRoot, err := bitwiseMerkleize(pubKeyChunks, uint64(len(pubKeyChunks)), uint64(len(pubKeyChunks)))
	if err != nil {
		return [32]byte{}, nil
	}
	fieldRoots[0] = pubKeyRoot[:]

	// Withdrawal credentials.
	fieldRoots[1] = validator.WithdrawalCredentials

	// Effective balance.
	effectiveBalanceBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(effectiveBalanceBuf, validator.EffectiveBalance)
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
	activationEligibilityBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(activationEligibilityBuf, validator.ActivationEligibilityEpoch)
	activationEligibilityRoot := bytesutil.ToBytes32(activationEligibilityBuf)
	fieldRoots[4] = activationEligibilityRoot[:]

	// Activation epoch.
	activationBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(activationBuf, validator.ActivationEpoch)
	activationRoot := bytesutil.ToBytes32(activationBuf)
	fieldRoots[5] = activationRoot[:]

	// Exit epoch.
	exitBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(exitBuf, validator.ExitEpoch)
	exitBufRoot := bytesutil.ToBytes32(exitBuf)
	fieldRoots[6] = exitBufRoot[:]

	// Withdrawable epoch.
	withdrawalBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(withdrawalBuf, validator.WithdrawableEpoch)
	withdrawalBufRoot := bytesutil.ToBytes32(withdrawalBuf)
	fieldRoots[7] = withdrawalBufRoot[:]

	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, nil
	}
	return root, nil
}
