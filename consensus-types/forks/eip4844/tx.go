package eip4844

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/protolambda/ztyp/codec"
)

var (
	errInvalidBlobTxType             = errors.New("invalid blob tx type")
	errInvalidVersionHashesVsKzg     = errors.New("invalid version hashes vs kzg")
	errInvalidNumBlobVersionedHashes = errors.New("invalid number of blob versioned hashes")
)

const (
	versionedHashesOffset = 258 // offset the versioned_hash_offset in a serialized blob tx
	messageLen            = 192 // size in bytes of "message" within the serialized blob tx
)

// TxPeekBlobVersionedHashes is from EIP-4844, and extracts the list of versioned hashes from the
// given blob tx.
//
// Format of the blob tx relevant to this function is as follows:
//   0: type (value should always be BlobTxType, 1 byte)
//   1: message offset (value should always be 69, 4 bytes)
//   5: ECDSA signature (65 bytes)
//   70: start of "message" (192 bytes)
//     258: start of the versioned hash offset within "message"  (4 bytes)
//   262-: rest of the tx following message
//
// TODO: unit tests, remove dependency on github.com/protolambda/ztyp
func TxPeekBlobVersionedHashes(tx []byte) ([]common.Hash, error) {
	// we start our reader at the versioned hash offset within the serialized tx
	if len(tx) < versionedHashesOffset {
		return nil, errors.New("blob tx invalid: too short")
	}
	if tx[0] != types.BlobTxType {
		return nil, errInvalidBlobTxType
	}
	dr := codec.NewDecodingReader(bytes.NewReader(tx[versionedHashesOffset:]), uint64(len(tx)-versionedHashesOffset))

	// read the offset to the versioned hashes
	var offset uint32
	offset, err := dr.ReadOffset()
	if err != nil {
		return nil, errors.Wrap(err, "could not read versioned hashes offset")
	}

	// Advance dr to the versioned hash list. We subtract messageLen from the offset here to
	// account for the fact that the offset is relative to the position of "message" (70) and we
	// are currently positioned at the end of it (262).
	skip := uint64(offset) - messageLen
	skipped, err := dr.Skip(skip)
	if err != nil {
		return nil, errors.Wrap(err, "could not skip to versioned hashes")
	}
	if skip != uint64(skipped) {
		return nil, fmt.Errorf("did not skip to versioned hashes. want %v got %v", skip, skipped)
	}

	// read the list of hashes one by one until we hit the end of the data
	hashes := []common.Hash{}
	tmp := make([]byte, 32)
	for dr.Scope() > 0 {
		if _, err = dr.Read(tmp); err != nil {
			return nil, errors.Wrap(err, "could not read versioned hashes")
		}
		var h common.Hash
		copy(h[:], tmp)
		hashes = append(hashes, h)
	}

	return hashes, nil
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
