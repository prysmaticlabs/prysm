package stategen

import "errors"

var errUnknownStateSummary = errors.New("unknown state summary")
var errUnknownArchivedState = errors.New("unknown archived state")
var errUnknownBoundaryState = errors.New("unknown boundary state")
var errUnknownState = errors.New("unknown state")
var errUnknownBlock = errors.New("unknown block")
