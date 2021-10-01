package eth

import (
	"fmt"

	ssz "github.com/ferranbt/fastssz"
)

// IMPORTANT
// The methods in this file are hand-written patches to the transaction type
// and ExecutionPayload.SizeSSZ. When you regenerate ssz methods make sure
// to delete the methods that conflict with what is in this file from
// generated.ssz.go or you will have a broken build.

// SizeSSZ returns the ssz encoded size in bytes for the ExecutionPayload object
// fastssz is not currently able to handle union types so it generates an incorrect
// implementation for this method.
func (e *ExecutionPayload) SizeSSZ() int {
	size := 508

	// Field (10) 'ExtraData'
	size += len(e.ExtraData)

	// Field (13) 'Transactions'
	for _, t := range e.Transactions {
		// offset
		size += 4
		// payload
		size += t.SizeSSZ()
	}

	return size
}

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
		t.TransactionOneof = &Transaction_OpaqueTransaction{buf[1:]}
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
func (t *Transaction) HashTreeRootWith(hh *ssz.Hasher) (error) {
	idx := hh.Index()

	switch t.TransactionOneof.(type) {
	case *Transaction_OpaqueTransaction:
		hh.PutBytes(t.GetOpaqueTransaction())
	default:
		return fmt.Errorf("can't HashTreeRootWith, Transaction oneof is using an unrecognized type option")
	}

	hh.Merkleize(idx)
	return nil
}
