package errutil

import "errors"

var (
	// AssignmentNotFoundErr represents the case in which a validator is not currently assigned in an epoch.
	AssignmentNotFoundErr = errors.New("assignments not found for validator")
)
