package eip4844

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/core/types"
)

var errInvalidBlobTxType = errors.New("invalid blob tx type")
var errInvalidVersionHashesVsKzg = errors.New("invalid version hashes vs kzg")

func TxPeekBlobVersionedHashes(tx []byte) ([][32]byte, error) {
	if tx[0] != 5 {
		return nil, errInvalidBlobTxType
	}
	offset := 1 + binary.BigEndian.Uint32(tx[1:5])
	hashesOffset := binary.BigEndian.Uint32(tx[offset+156 : offset+160])
	hashes := make([][32]byte, (uint32(len(tx))-hashesOffset)/32)
	for i := hashesOffset; i < uint32(len(tx)); i += 32 {
		var hash [32]byte
		copy(hash[:], tx[i:i+32])
		hashes = append(hashes, hash)
	}
	return hashes, nil
}

func VerifyKzgsAgainstTxs(txs [][]byte, blogKzgs [][48]byte) error {
	versionedHashes := make([][32]byte, 0)
	for _, tx := range txs {
		hs, err := TxPeekBlobVersionedHashes(tx)
		if err != nil {
			return err
		}
		versionedHashes = append(versionedHashes, hs...)
	}
	for i, kzg := range blogKzgs {
		h := types.KZGCommitment(kzg).ComputeVersionedHash()
		if h != versionedHashes[i] {
			return errInvalidVersionHashesVsKzg
		}
	}
	return nil
}
