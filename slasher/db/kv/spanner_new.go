package kv

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

var errWrongSize = errors.New("wrong data length for min max span byte array")
var highestObservedValidatorIdx uint64

// GetValidatorSpan unmarshal a span from an encoded, flattened array.
func (es *EpochStore) GetValidatorSpan(ctx context.Context, idx uint64) (types.Span, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.getValidatorSpan")
	defer span.End()

	r := types.Span{}
	if len(es.Spans)%spannerEncodedLength != 0 {
		return r, errWrongSize
	}
	origLength := uint64(len(es.Spans)) / spannerEncodedLength
	requestedLength := idx + 1
	if origLength < requestedLength {
		return r, nil
	}
	cursor := idx * spannerEncodedLength
	r.MinSpan = bytesutil.FromBytes2(es.Spans[cursor : cursor+2])
	r.MaxSpan = bytesutil.FromBytes2(es.Spans[cursor+2 : cursor+4])
	sigB := [2]byte{}
	copy(sigB[:], es.Spans[cursor+4:cursor+6])
	r.SigBytes = sigB
	r.HasAttested = bytesutil.ToBool(es.Spans[cursor+6])
	return r, nil
}

// SetValidatorSpan marshal a validator span into an encoded, flattened array.
func (es *EpochStore) SetValidatorSpan(ctx context.Context, idx uint64, newSpan types.Span) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.setValidatorSpan")
	defer span.End()

	if len(es.Spans)%spannerEncodedLength != 0 {
		return errors.New("wrong data length for min max span byte array")
	}
	if highestObservedValidatorIdx < idx {
		highestObservedValidatorIdx = idx
	}
	if len(es.Spans) == 0 {
		requestedLength := highestObservedValidatorIdx*spannerEncodedLength + spannerEncodedLength
		b := make([]byte, requestedLength, requestedLength)
		es.Spans = b

	}
	cursor := idx * spannerEncodedLength
	endCursor := cursor + spannerEncodedLength
	spansLength := uint64(len(es.Spans))
	if endCursor > spansLength {
		diff := endCursor - spansLength
		b := make([]byte, diff, diff)
		es.Spans = append(es.Spans, b...)
	}
	enc := marshalSpan(newSpan)
	copy(es.Spans[cursor:], enc)

	return nil
}

// EpochSpans accepts epoch and returns the corresponding spans byte array
// for slashing detection.
// Returns span byte array, and error in case of db error.
// returns empty byte array if no entry for this epoch exists in db.
func (db *Store) EpochSpans(ctx context.Context, epoch uint64) (EpochStore, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.EpochSpans")
	defer span.End()

	var err error
	var spans []byte
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(validatorsMinMaxSpanBucketNew)
		if b == nil {
			return nil
		}
		spans = b.Get(bytesutil.Bytes8(epoch))
		return nil
	})
	if spans == nil {
		spans = []byte{}
	}
	es := EpochStore{Spans: spans}
	return es, err
}

// SaveEpochSpans accepts a epoch and span byte array and writes it to disk.
func (db *Store) SaveEpochSpans(ctx context.Context, epoch uint64, es EpochStore) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SaveEpochSpans")
	defer span.End()

	if len(es.Spans)%spannerEncodedLength != 0 {
		return errWrongSize
	}
	return db.update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(validatorsMinMaxSpanBucketNew)
		if err != nil {
			return err
		}
		return b.Put(bytesutil.Bytes8(epoch), es.Spans)
	})
}
