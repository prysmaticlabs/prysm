package eth

import (
	"fmt"

	"github.com/pkg/errors"

	ssz "github.com/ferranbt/fastssz"
)

var ErrUnexpectedTransactionType = errors.New("Unexpected oneof type for ExecutionPayload.Transactions")

// IMPORTANT
// The methods in this file are hand-written patches to the Transaction type.

// MarshalSSZ ssz marshals the Transaction object
func (t *Transaction) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(t)
}

// MarshalSSZTo ssz marshals the Transaction object to a target array
func (t *Transaction) MarshalSSZTo(buf []byte) ([]byte, error) {
	// 1 byte to indiciate the type (spec calls this the selector)
	var selector byte
	switch t.TransactionOneof.(type) {
	case *Transaction_OpaqueTransaction:
		ot := t.GetOpaqueTransaction()
		if len(ot) > 1048576 {
			return nil, ssz.ErrBytesLength
		}
		// selector will be zero'd out by slice initialization
		buf = append(buf, selector)
		buf = append(buf, ot...)
		return buf, nil
	}

	return nil, fmt.Errorf("can't MarshalSSZTo, Transaction oneof is using an unrecognized type option")
}

// UnmarshalSSZ ssz unmarshals the Transaction object
func (t *Transaction) UnmarshalSSZ(buf []byte) error {
	size := uint64(len(buf))
	if size == 0 {
		return fmt.Errorf("Can't unmarshal empty slice")
	}
	selector := buf[0]
	switch selector {
	case 0b00000000:
		if size > 1048576+1 {
			return ssz.ErrSize
		}
		ob := make([]byte, len(buf[1:]))
		copy(ob, buf[1:])
		t.TransactionOneof = &Transaction_OpaqueTransaction{ob}
	}

	return nil
}

// SizeSSZ returns the ssz encoded size in bytes for the Transaction object
func (t *Transaction) SizeSSZ() int {
	switch t.TransactionOneof.(type) {
	case *Transaction_OpaqueTransaction:
		// each case should add an extra byte for the type selector
		return len(t.GetOpaqueTransaction()) + 1
	}
	return 0
}

// HashTreeRoot ssz hashes the Transaction object
func (t *Transaction) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(t)
}

// HashTreeRootWith ssz hashes the Transaction object with a hasher
func (t *Transaction) HashTreeRootWith(hh *ssz.Hasher) error {
	idx := hh.Index()

	var selector byte
	switch t.TransactionOneof.(type) {
	case *Transaction_OpaqueTransaction:
		opaque := t.GetOpaqueTransaction()

		byteLen := uint64(len(opaque))
		if byteLen > 1048576 {
			return ssz.ErrIncorrectListSize
		}

		// zero pad if not aligned to chunk size
		if byteLen%32 != 0 {
			zeroes := make([]byte, 32-(byteLen%32))
			opaque = append(opaque, zeroes...)
		}
		hh.Append(opaque)
		hh.MerkleizeWithMixin(idx, byteLen, 1048576/32)
		hh.MerkleizeMixInSelector(idx, selector)
	default:
		return fmt.Errorf("can't HashTreeRootWith, Transaction oneof is using an unrecognized type option")
	}
	return nil
}

func OpaqueTransactions(ep *ExecutionPayload) ([][]byte, error) {
	ots := make([][]byte, len(ep.Transactions))
	for _, t := range ep.Transactions {
		switch t.TransactionOneof.(type) {
		case *Transaction_OpaqueTransaction:
			ots = append(ots, t.GetOpaqueTransaction())
		default:
			return nil, ErrUnexpectedTransactionType
		}
	}
	return ots, nil
}
