package stategen

import "errors"

var errUnknownBoundaryState = errors.New("unknown boundary state")
var errUnknownState = errors.New("unknown state")
var errUnknownBlock = errors.New("unknown block")

// errNilState returns when we have obtained a nil state from stategen
var errNilState = errors.New("nil state from stategen")
