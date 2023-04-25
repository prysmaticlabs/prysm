package consensus_types

import "errors"

// ErrNilObjectWrapped is returned in a constructor when the underlying object is nil.
var ErrNilObjectWrapped = errors.New("attempted to wrap nil object")
