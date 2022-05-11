package state_native

import "github.com/pkg/errors"

// ErrNilField returns when the inner field in the Beacon state is nil
var ErrNilField = errors.New("nil field")
