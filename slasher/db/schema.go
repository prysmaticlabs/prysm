package db

import (
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

var (
	// Slasher
	historicIndexedAttestationsBucket = []byte("historic-indexed-attestations-bucket")
	historicBlockHeadersBucket        = []byte("historic-block-headers-bucket")
	indexedAttestationsIndicesBucket  = []byte("indexed-attestations-indices-bucket")
)

func encodeEpochValidatorID(epoch uint64, validatorID uint64) []byte {
	return append(bytesutil.Bytes8(epoch), bytesutil.Bytes8(validatorID)...)
}

func encodeEpochValidatorIDSig(epoch uint64, validatorID uint64, sig []byte) []byte {
	return append(append(bytesutil.Bytes8(epoch), bytesutil.Bytes8(validatorID)...), sig...)
}

func encodeEpochSig(epoch uint64, sig []byte) []byte {
	return append(bytesutil.Bytes8(epoch), sig...)
}
