package eip4844

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/protolambda/ztyp/codec"
)

var errInvalidBlobTxType = errors.New("invalid blob tx type")
var errInvalidVersionHashesVsKzg = errors.New("invalid version hashes vs kzg")

func TxPeekBlobVersionedHashes(tx []byte) ([][32]byte, error) {
	if tx[0] != types.BlobTxType {
		return nil, errInvalidBlobTxType
	}
	sbt := types.SignedBlobTx{}
	if err := sbt.Deserialize(codec.NewDecodingReader(bytes.NewReader(tx[1:]), uint64(len(tx)-1))); err != nil {
		return nil, err
	}
	hashes := make([][32]byte, len(sbt.Message.BlobVersionedHashes))
	for _, b := range sbt.Message.BlobVersionedHashes {
		var hash [32]byte
		copy(hash[:], b[:])
		hashes = append(hashes, hash)
	}
	return hashes, nil
}

func VerifyKzgsAgainstTxs(txs [][]byte, blogKzgs [][48]byte) error {
	versionedHashes := make([][32]byte, 0)
	for _, tx := range txs {
		if tx[0] == types.BlobTxType {
			hs, err := TxPeekBlobVersionedHashes(tx)
			if err != nil {
				return err
			}
			versionedHashes = append(versionedHashes, hs...)
		}
	}
	// TODO(inphi): modify validation spec to handle when len(blob_txs) > len(blobKzgs)
	for i, kzg := range blogKzgs {
		h := types.KZGCommitment(kzg).ComputeVersionedHash()
		if h != versionedHashes[i] {
			return errInvalidVersionHashesVsKzg
		}
	}
	return nil
}
