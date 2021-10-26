package kv

import "errors"

var ErrNotFound = errors.New("not found in db")
var ErrNotFoundOriginCheckpoint = WrapDBError(ErrNotFound, "OriginCheckpointRoot")
var ErrNotFoundFinalizedCheckpoint = WrapDBError(ErrNotFound, "FinalizedCheckpoint")

func WrapDBError(e error, outer string) error {
	return DBError{
		Wraps: e,
		Outer: errors.New(outer),
	}
}

type DBError struct {
	Wraps error
	Outer error
}

func (e DBError) Error() string {
	es := e.Outer.Error()
	if e.Wraps != nil {
		es += ": " + e.Wraps.Error()
	}
	return es
}

func (e DBError) Unwrap() error {
	return e.Wraps
}
