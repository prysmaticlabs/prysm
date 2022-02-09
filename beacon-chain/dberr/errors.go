package dberr

import "github.com/pkg/errors"

var ErrNotFound = errors.New("not found in database")
var ErrStateNotFound = errors.Wrap(ErrNotFound, "state not found")
