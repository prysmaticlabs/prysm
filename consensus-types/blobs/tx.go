package blobs

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/protolambda/ztyp/codec"
)

var (
	errInvalidBlobTxType             = errors.New("invalid blob tx type")
	errInvalidVersionHashesVsKzg     = errors.New("invalid version hashes vs kzg")
	errInvalidNumBlobVersionedHashes = errors.New("invalid number of blob versioned hashes")
)

// TxPeekBlobVersionedHashes returns the versioned hashes of a blob tx.
// Spec code:
// def tx_peek_blob_versioned_hashes(opaque_tx: Transaction) -> Sequence[VersionedHash]:
//    assert opaque_tx[0] == BLOB_TX_TYPE
//    message_offset = 1 + uint32.decode_bytes(opaque_tx[1:5])
//    # field offset: 32 + 8 + 32 + 32 + 8 + 4 + 32 + 4 + 4 = 156
//    blob_versioned_hashes_offset = (
//        message_offset
//        + uint32.decode_bytes(opaque_tx[(message_offset + 156):(message_offset + 160)])
//    )
//    return [
//        VersionedHash(opaque_tx[x:(x + 32)])
//        for x in range(blob_versioned_hashes_offset, len(opaque_tx), 32)
//    ]
func TxPeekBlobVersionedHashes(tx []byte) ([]common.Hash, error) {
	if tx[0] != types.BlobTxType {
		return nil, errInvalidBlobTxType
	}
	// TODO remove geth/ztyp dep
	sbt := types.SignedBlobTx{}
	if err := sbt.Deserialize(codec.NewDecodingReader(bytes.NewReader(tx[1:]), uint64(len(tx)-1))); err != nil {
		return nil, fmt.Errorf("%w: unable to decode Blob Tx", err)
	}
	return sbt.Message.BlobVersionedHashes, nil
}

// VerifyKzgsAgainstTxs verifies blob kzgs against the transactions.
// Spec code:
// def verify_kzg_commitments_against_transactions(transactions: Sequence[Transaction],
//                                                kzg_commitments: Sequence[KZGCommitment]) -> bool:
//    all_versioned_hashes = []
//    for tx in transactions:
//        if tx[0] == BLOB_TX_TYPE:
//            all_versioned_hashes += tx_peek_blob_versioned_hashes(tx)
//    return all_versioned_hashes == [kzg_commitment_to_versioned_hash(commitment) for commitment in kzg_commitments]
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
