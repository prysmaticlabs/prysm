package kv

import "github.com/pkg/errors"

// errDeleteFinalized is raised when we attempt to delete a finalized block/state
var errDeleteFinalized = errors.New("cannot delete finalized block or state")

// ErrNotFound can be used directly, or as a wrapped DBError, whenever a db method needs to
// indicate that a value couldn't be found.
var ErrNotFound = errors.New("not found in db")
var ErrNotFoundState = errors.Wrap(ErrNotFound, "state not found")

// ErrNotFoundOriginBlockRoot is an error specifically for the origin block root getter
var ErrNotFoundOriginBlockRoot = errors.Wrap(ErrNotFound, "OriginBlockRoot")