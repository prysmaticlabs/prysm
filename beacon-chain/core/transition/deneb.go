package transition

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	blobCommitmentVersionKZG  uint8 = 0x01
	blobTxType                      = 3
	precompileInputLength           = 192
	blobVersionedHashesOffset       = 258 // position of blob_versioned_hashes offset in a serialized blob tx, see TxPeekBlobVersionedHashes
)

// kzgToVersionedHash implements kzg_to_versioned_hash from EIP-4844
func kzgToVersionedHash(kzg []byte) (h [32]byte) {
	h = sha256.Sum256(kzg)
	h[0] = blobCommitmentVersionKZG
	return
}

// verifyKZGCommitmentsAgainstTransactions implements verify_kzg_commitments_against_transactions
// from the EIP-4844 consensus spec:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/eip4844/beacon-chain.md#verify_kzg_commitments_against_transactions
func verifyKZGCommitmentsAgainstTransactions(transactions, kzgCommitments [][]byte) error {
	var versionedHashes [][32]byte
	for _, tx := range transactions {
		if tx[0] == blobTxType {
			v, err := TxPeekBlobVersionedHashes(tx)
			if err != nil {
				return err
			}
			versionedHashes = append(versionedHashes, v...)
		}
	}
	if len(kzgCommitments) != len(versionedHashes) {
		return fmt.Errorf("invalid number of blob versioned hashes: %v vs %v", len(kzgCommitments), len(versionedHashes))
	}
	for i := 0; i < len(kzgCommitments); i++ {
		h := kzgToVersionedHash(kzgCommitments[i])
		if h != versionedHashes[i] {
			return errors.New("invalid version hashes vs kzg")
		}
	}
	return nil
}

// txPeekBlobVersionedHashes implements tx_peek_blob_versioned_hashes from EIP-4844 consensus spec:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/eip4844/beacon-chain.md#tx_peek_blob_versioned_hashes
//
// Format of the blob tx relevant to this function is as follows:
//
//		0: type (value should always be blobTxType)
//		1: message offset: 4 bytes
//		5: ECDSA signature: 65 bytes
//		70: start of "message": 192 bytes
//			70: chain_id: 32 bytes
//			102: nonce: 8 bytes
//			110: priority_fee_per_gas: 32 bytes
//			142: max_basefee_per_gas: 32 bytes
//			174: gas: 8 bytes
//			182: to: 4 bytes - offset (relative to "message")
//			186: value: 32 bytes
//			218: data: 4 bytes - offset (relative to "message")
//			222: access_list: 4 bytes - offset (relative to "message")
//			226: max_fee_per_data_gas: 32 bytes
//			258: blob_versioned_hashes: 4 bytes - offset (relative to "message")
//	     262: start of dynamic data of "message"
//
// This function does not fully verify the encoding of the provided tx, but will sanity-check the tx type,
// and will never panic on malformed inputs.
func TxPeekBlobVersionedHashes(tx []byte) ([][32]byte, error) {
	// we start our reader at the versioned hash offset within the serialized tx
	if len(tx) < blobVersionedHashesOffset+4 {
		return nil, errors.New("blob tx invalid: too short")
	}
	if tx[0] != blobTxType {
		return nil, errors.New("invalid blob tx type")
	}
	offset := uint64(binary.LittleEndian.Uint32(tx[blobVersionedHashesOffset:blobVersionedHashesOffset+4])) + 70
	if offset > uint64(len(tx)) {
		return nil, errors.New("offset to versioned hashes is out of bounds")
	}
	hashBytesLen := uint64(len(tx)) - offset
	if hashBytesLen%32 != 0 {
		return nil, errors.New("expected trailing data starting at versioned-hashes offset to be a multiple of 32 bytes")
	}
	hashes := make([][32]byte, hashBytesLen/32)
	for i := range hashes {
		copy(hashes[i][:], tx[offset:offset+32])
		offset += 32
	}
	return hashes, nil
}
