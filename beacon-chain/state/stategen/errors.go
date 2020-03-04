package stategen

import "errors"

var errUnknownStateSummary = errors.New("unknown state summary")
var errUnknownArchivedState = errors.New("unknown archived state")
var errUnknownBoundaryState = errors.New("unknown boundary state")
var errUnknownBoundaryRoot = errors.New("unknown boundary root")
var errUnknownState = errors.New("unknown state")
var errUnknownBlock = errors.New("unknown block")
var errSlotNonArchivedPoint = errors.New("slot is not an archived point index")
