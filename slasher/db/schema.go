package db

import (
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

var (
	// Slasher
	historicAttestationsBucket = []byte("historic-attestations-bucket")
	historicBlockHeadersBucket = []byte("historic-block-headers-bucket")
)

func encodeEpochValidatorID(epoch uint64, validatorID uint64) []byte {
	return append(bytesutil.Bytes8(epoch), bytesutil.Bytes8(validatorID)...)
}

func encodeEpochValidatorIDSig(epoch uint64, validatorID uint64, sig []byte) []byte {
	return append(append(bytesutil.Bytes8(epoch), bytesutil.Bytes8(validatorID)...), sig...)
}
