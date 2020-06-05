package types

import (
	"errors"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
func NewEpochStore(spans []byte) (*EpochStore, error) {
	if len(spans)%int(SpannerEncodedLength) != 0 {
		return &EpochStore{}, ErrWrongSize
	}
	es := &EpochStore{
		spans: spans,
	}
	return es, nil
}

// GetValidatorSpan unmarshal a span from an encoded, flattened array.
func (es *EpochStore) GetValidatorSpan(idx uint64) (Span, error) {
	r := Span{}
	if len(es.spans)%int(SpannerEncodedLength) != 0 {
		return r, ErrWrongSize
	}
	origLength := uint64(len(es.spans)) / SpannerEncodedLength
	requestedLength := idx + 1
	if origLength < requestedLength {
		return r, nil
	}
	cursor := idx * SpannerEncodedLength
	r.MinSpan = bytesutil.FromBytes2(es.spans[cursor : cursor+2])
	r.MaxSpan = bytesutil.FromBytes2(es.spans[cursor+2 : cursor+4])
	sigB := [2]byte{}
	copy(sigB[:], es.spans[cursor+4:cursor+6])
	r.SigBytes = sigB
	r.HasAttested = bytesutil.ToBool(es.spans[cursor+6])
	return r, nil
}

// SetValidatorSpan marshal a validator span into an encoded, flattened array.
func (es *EpochStore) SetValidatorSpan(idx uint64, newSpan Span) (*EpochStore, error) {
	spansLen := uint64(len(es.spans))
	if spansLen%SpannerEncodedLength != 0 {
		return nil, errors.New("wrong data length for min max span byte array")
	}
	if es.highestObservedIdx < idx {
		es.highestObservedIdx = idx
	}
	if spansLen == 0 {
		requestedLength := es.highestObservedIdx*SpannerEncodedLength + SpannerEncodedLength
		es.spans = make([]byte, requestedLength)
	}
	cursor := idx * SpannerEncodedLength
	endCursor := cursor + SpannerEncodedLength
	spansLength := uint64(len(es.spans))
	if endCursor > spansLength {
		diff := endCursor - spansLength
		b := make([]byte, diff)
		es.spans = append(es.spans, b...)
	}
	enc := newSpan.Marshal()
	copy(es.spans[cursor:], enc)

	return es, nil
}

// HighestObservedIdx returns the highest idx the EpochStore has been used for.
func (es *EpochStore) HighestObservedIdx() uint64 {
	return es.highestObservedIdx
}

// Bytes returns the underlying bytes of an EpochStore.
func (es *EpochStore) Bytes() []byte {
	return es.spans
}

// ToMap is a helper function to convert an epoch store to a map, mainly used for testing.
func (es *EpochStore) ToMap() (map[uint64]Span, error) {
	spanMap := make(map[uint64]Span)
	var err error
	for i := uint64(0); i < es.highestObservedIdx; i++ {
		spanMap[i], err = es.GetValidatorSpan(i)
		if err != nil {
			return nil, err
		}
	}
	return spanMap, nil
}
