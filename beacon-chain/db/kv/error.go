package kv

import "errors"

// ErrNotFound can be used directly, or as a wrapped DBError, whenever a db method needs to
// indicate that a value couldn't be found.
var ErrNotFound = errors.New("not found in db")

// ErrNotFoundOriginBlockRoot is an error specifically for the origin block root getter
var ErrNotFoundOriginBlockRoot = WrapDBError(ErrNotFound, "OriginBlockRoot")

// WrapDBError wraps an error in a DBError. See commentary on DBError for more context.
func WrapDBError(e error, outer string) error {
	return DBError{
		Wraps: e,
		Outer: errors.New(outer),
	}
}

// DBError implements the Error method so that it can be asserted as an error.
// The Unwrap method supports error wrapping, enabling it to be used with errors.Is/As.
// The primary use case is to make it simple for database methods to return errors
// that wrap ErrNotFound, allowing calling code to check for "not found" errors
// like: `error.Is(err, ErrNotFound)`. This is intended to improve error handling
// in db lookup methods that need to differentiate between a missing value and some
// other database error. for more background see:
// https://go.dev/blog/go1.13-errors
type DBError struct {
	Wraps error
	Outer error
}

// Error satisfies the error interface, so that DBErrors can be used anywhere that
// expects an `error`.
func (e DBError) Error() string {
	es := e.Outer.Error()
	if e.Wraps != nil {
		es += ": " + e.Wraps.Error()
	}
	return es
}

// Unwrap is used by the errors package Is and As methods.
func (e DBError) Unwrap() error {
	return e.Wraps
}
