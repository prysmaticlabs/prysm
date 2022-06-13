package eip4844

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/protolambda/ztyp/codec"
)

var (
	errInvalidBlobTxType             = errors.New("invalid blob tx type")
	errInvalidVersionHashesVsKzg     = errors.New("invalid version hashes vs kzg")
	errInvalidNumBlobVersionedHashes = errors.New("invalid number of blob versioned hashes")
)

func TxPeekBlobVersionedHashes(tx []byte) ([]common.Hash, error) {
	if tx[0] != types.BlobTxType {
		return nil, errInvalidBlobTxType
	}
	sbt := types.SignedBlobTx{}
	if err := sbt.Deserialize(codec.NewDecodingReader(bytes.NewReader(tx[1:]), uint64(len(tx)-1))); err != nil {
		return nil, err
	}
	return sbt.Message.BlobVersionedHashes, nil
}

func VerifyKzgsAgainstTxs(txs [][]byte, blobKzgs [][48]byte) error {
	versionedHashes := make([]common.Hash, 0)
	for _, tx := range txs {
		if tx[0] == types.BlobTxType {
			hs, err := TxPeekBlobVersionedHashes(tx)
			if err != nil {
				return err
			}
			versionedHashes = append(versionedHashes, hs...)
		}
	}
	if len(blobKzgs) != len(versionedHashes) {
		return errInvalidNumBlobVersionedHashes
	}
	for i, kzg := range blobKzgs {
		h := types.KZGCommitment(kzg).ComputeVersionedHash()
		if h != versionedHashes[i] {
			return errInvalidVersionHashesVsKzg
		}
	}
	return nil
}
