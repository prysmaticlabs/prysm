package stategen

import "errors"

var errNonArchivedPoint = errors.New("unable to store non archived point state")
var errUnknownColdSummary = errors.New("unknown cold state summary")
var errUnknownHotSummary = errors.New("unknown hot state summary")
var errUnknownArchivedState = errors.New("unknown archived state")
var errUnknownBoundaryState = errors.New("unknown boundary state")
var errUnknownBoundaryRoot = errors.New("unknown boundary root")
var errUnknownState = errors.New("unknown state")
var errUnknownBlock = errors.New("unknown block")
