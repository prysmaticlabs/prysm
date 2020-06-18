package types

import (
	"github.com/pkg/errors"
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
	spansLen := uint64(len(spans))
	if spansLen%SpannerEncodedLength != 0 {
		return &EpochStore{}, ErrWrongSize
	}
	highestIdx := spansLen / SpannerEncodedLength
	if highestIdx > 0 {
		// Minus one here since validators are 0 index.
		highestIdx--
	}
	es := &EpochStore{
		spans:              spans,
		highestObservedIdx: highestIdx,
	}
	return es, nil
}

// GetValidatorSpan unmarshal a span from an encoded, flattened array.
func (es *EpochStore) GetValidatorSpan(idx uint64) (Span, error) {
	spansLen := uint64(len(es.spans))
	if spansLen%SpannerEncodedLength != 0 {
		return Span{}, ErrWrongSize
	}
	if idx > es.highestObservedIdx {
		return Span{}, nil
	}
	origLength := uint64(len(es.spans)) / SpannerEncodedLength
	requestedLength := idx + 1
	if origLength < requestedLength {
		return Span{}, nil
	}
	cursor := idx * SpannerEncodedLength
	unmarshaledSpan, err := UnmarshalSpan(es.spans[cursor : cursor+SpannerEncodedLength])
	if err != nil {
		return Span{}, errors.Wrap(err, "could not unmarshal span")
	}
	return unmarshaledSpan, nil
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
	copy(es.spans[cursor:], newSpan.Marshal())
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
	spanMap := make(map[uint64]Span, es.highestObservedIdx)
	var err error
	spansLen := uint64(len(es.spans)) / SpannerEncodedLength
	if spansLen > 0 {
		spansLen--
	} else if spansLen == 0 {
		return spanMap, nil
	}
	for i := uint64(0); i <= spansLen; i++ {
		spanMap[i], err = es.GetValidatorSpan(i)
		if err != nil {
			return nil, err
		}
	}
	return spanMap, nil
}

// EpochStoreFromMap is a helper function to turn a map into a epoch store, mainly used for testing.
func EpochStoreFromMap(spanMap map[uint64]Span) (*EpochStore, error) {
	var err error
	es, err := NewEpochStore([]byte{})
	if err != nil {
		return nil, err
	}
	for k, v := range spanMap {
		if es, err = es.SetValidatorSpan(k, v); err != nil {
			return nil, err
		}
	}
	return es, nil
}
