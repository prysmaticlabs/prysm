package kv

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

// EpochStore defines an implementation of the slasher data access interface
// using byte array as data source to extract and put validator spans into.
type EpochStore []byte

// ErrWrongSize appears when attempting to use epoch store byte array with size that
// is not a multiple of spanner encoded length.
var ErrWrongSize = errors.New("wrong data length for min max span byte array")
var highestObservedValidatorIdx uint64

// NewEpochStore initialize epoch store from a byte array
// returns error if byte length is not a multiple of encoded spanner length.
func NewEpochStore(spans []byte) (EpochStore, error) {
	if len(spans)%spannerEncodedLength != 0 {
		return nil, ErrWrongSize
	}
	es := EpochStore{}
	es = spans
	return es, nil
}

// GetValidatorSpan unmarshal a span from an encoded, flattened array.
func (es EpochStore) GetValidatorSpan(ctx context.Context, idx uint64) (types.Span, error) {
	r := types.Span{}
	if len(es)%spannerEncodedLength != 0 {
		return r, ErrWrongSize
	}
	origLength := uint64(len(es)) / spannerEncodedLength
	requestedLength := idx + 1
	if origLength < requestedLength {
		return r, nil
	}
	cursor := idx * spannerEncodedLength
	r.MinSpan = bytesutil.FromBytes2(es[cursor : cursor+2])
	r.MaxSpan = bytesutil.FromBytes2(es[cursor+2 : cursor+4])
	sigB := [2]byte{}
	copy(sigB[:], es[cursor+4:cursor+6])
	r.SigBytes = sigB
	r.HasAttested = bytesutil.ToBool(es[cursor+6])
	return r, nil
}

// SetValidatorSpan marshal a validator span into an encoded, flattened array.
func (es *EpochStore) SetValidatorSpan(ctx context.Context, idx uint64, newSpan types.Span) error {
	if len(*es)%spannerEncodedLength != 0 {
		return errors.New("wrong data length for min max span byte array")
	}
	if highestObservedValidatorIdx < idx {
		highestObservedValidatorIdx = idx
	}
	if len(*es) == 0 {
		requestedLength := highestObservedValidatorIdx*spannerEncodedLength + spannerEncodedLength
		*es = make([]byte, requestedLength, requestedLength)
	}
	cursor := idx * spannerEncodedLength
	endCursor := cursor + spannerEncodedLength
	spansLength := uint64(len(*es))
	if endCursor > spansLength {
		diff := endCursor - spansLength
		b := make([]byte, diff, diff)
		*es = append(*es, b...)
	}
	enc := marshalSpan(newSpan)
	ba := *es
	copy(ba[cursor:], enc)

	return nil
}
