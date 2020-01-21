package protoarray

import "errors"

var unknownFinalizedRoot = errors.New("unknown finalized root")
var unknownJustifiedRoot = errors.New("unknown justified root")
var invalidNodeIndex = errors.New("node index is invalid")
var invalidJustifiedIndex = errors.New("justified index is invalid")
var invalidBestDescendantIndex = errors.New("best descendant index is invalid")
var invalidParentDelta = errors.New("parent delta is invalid")
var invalidNodeDelta = errors.New("node delta is invalid")
var invalidDeltaLength = errors.New("delta length is invalid")
var invalidBestNode = errors.New("best node is not valid due to filter")
