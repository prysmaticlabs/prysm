package db

import (
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

var (
	// Protector
	historicProposalsBucket = []byte("historic-proposals-bucket")
	// In order to quickly detect surround and surrounded attestations we need to store
	// the min and max span for each validator for each epoch.
	// see https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
	validatorsMinMaxSpanBucket = []byte("validators-min-max-span-bucket")
)

func encodeEpochValidatorID(epoch uint64, validatorIdx uint64) []byte {
	return append(bytesutil.Bytes8(epoch), bytesutil.Bytes8(validatorIdx)...)
}

func encodeEpochSig(targetEpoch uint64, sig []byte) []byte {
	return append(bytesutil.Bytes8(targetEpoch), sig...)
}
