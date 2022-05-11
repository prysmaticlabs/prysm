package v2

import "github.com/pkg/errors"

// ErrNilInnerState returns when the inner state is nil and no copy set or get
// operations can be performed on state.
var ErrNilInnerState = errors.New("nil inner state")

// ErrNilField returns when the inner field in the Beacon state is nil
var ErrNilField = errors.New("nil field")
