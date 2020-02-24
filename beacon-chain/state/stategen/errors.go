package stategen

import "errors"

var errNonArchivedPoint = errors.New("unable to store non archived point state")
var errUnknownColdSummary = errors.New("unknown cold state summary")
var errUnknownArchivedState = errors.New("unknown archived state")