package kv

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

// EpochStore defines an implementation of the slasher data access interface
// using byte array as data source to extract and put validator spans into.
type EpochStore struct {
	spans              []byte
	highestObservedIdx uint64
}

// ErrWrongSize appears when attempting to use epoch store byte array with size that
// is not a multiple of spanner encoded length.
var ErrWrongSize = errors.New("wrong data length for min max span byte array")

// NewEpochStore initialize epoch store from a byte array
// returns error if byte length is not a multiple of encoded spanner length.
func NewEpochStore(spans []byte) (EpochStore, error) {
	if len(spans)%spannerEncodedLength != 0 {
		return EpochStore{}, ErrWrongSize
	}
	es := EpochStore{
		spans: spans,
	}
	return es, nil
}

// GetValidatorSpan unmarshal a span from an encoded, flattened array.
func (es EpochStore) GetValidatorSpan(ctx context.Context, idx uint64) (types.Span, error) {
	r := types.Span{}
	if len(es.spans)%spannerEncodedLength != 0 {
		return r, ErrWrongSize
	}
	origLength := uint64(len(es.spans)) / spannerEncodedLength
	requestedLength := idx + 1
	if origLength < requestedLength {
		return r, nil
	}
	cursor := idx * spannerEncodedLength
	r.MinSpan = bytesutil.FromBytes2(es.spans[cursor : cursor+2])
	r.MaxSpan = bytesutil.FromBytes2(es.spans[cursor+2 : cursor+4])
	sigB := [2]byte{}
	copy(sigB[:], es.spans[cursor+4:cursor+6])
	r.SigBytes = sigB
	r.HasAttested = bytesutil.ToBool(es[cursor+6])
	return r, nil
}

// SetValidatorSpan marshal a validator span into an encoded, flattened array.
func (es EpochStore) SetValidatorSpan(ctx context.Context, idx uint64, newSpan types.Span) error {
	if len(es.spans)%spannerEncodedLength != 0 {
		return errors.New("wrong data length for min max span byte array")
	}
	if es.highestObservedIdx < idx {
		es.highestObservedIdx = idx
	}
	if len(es.spans) == 0 {
		requestedLength := es.highestObservedIdx*spannerEncodedLength + spannerEncodedLength
		es.spans = make([]byte, requestedLength)
	}
	cursor := idx * spannerEncodedLength
	endCursor := cursor + spannerEncodedLength
	spansLength := uint64(len(es.spans))
	if endCursor > spansLength {
		diff := endCursor - spansLength
		b := make([]byte, diff)
		es.spans = append(es.spans, b...)
	}
	enc := marshalSpan(newSpan)
	ba := es.spans
	copy(ba[cursor:], enc)

	return nil
}

// ToMap is a helper function to convert an epoch store to a map, mainly used for testing.
func (es *EpochStore) ToMap() (map[uint64]types.Span, error) {
	spanMap := make(map[uint64]types.Span)
	var err error
	for i := uint64(0); i < es.highestObservedIdx; i++ {
		spanMap[i], err = es.GetValidatorSpan(context.Background(), i)
		if err != nil {
			return nil, err
		}
	}
	return spanMap, nil
}
