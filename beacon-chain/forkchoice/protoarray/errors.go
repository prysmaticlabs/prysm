package protoarray

import "errors"

var errNilNode = errors.New("invalid nil node")
var errInvalidBalance = errors.New("invalid node balance")
var errUnknownFinalizedRoot = errors.New("unknown finalized root")
var errUnknownJustifiedRoot = errors.New("unknown justified root")
var errInvalidJustifiedIndex = errors.New("justified index is invalid")
var errInvalidBestChildIndex = errors.New("best child index is invalid")
var errInvalidBestDescendantIndex = errors.New("best descendant index is invalid")
var errInvalidParentDelta = errors.New("parent delta is invalid")
var errInvalidNodeDelta = errors.New("node delta is invalid")
var errInvalidDeltaLength = errors.New("delta length is invalid")
