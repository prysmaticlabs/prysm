package protoarray

import "errors"

var errNilNode = errors.New("invalid nil or unknown node")
var errInvalidBalance = errors.New("invalid node balance")
var errInvalidProposerBoostRoot = errors.New("invalid proposer boost root")
var errUnknownFinalizedRoot = errors.New("unknown finalized root")
var errUnknownJustifiedRoot = errors.New("unknown justified root")
var errInvalidJustifiedIndex = errors.New("justified index is invalid")
var errInvalidBestChildIndex = errors.New("best child index is invalid")
var errInvalidBestDescendantIndex = errors.New("best descendant index is invalid")
var errInvalidOptimisticStatus = errors.New("invalid optimistic status")
var errInvalidParentDelta = errors.New("parent delta is invalid")
var errInvalidNodeDelta = errors.New("node delta is invalid")
var errInvalidDeltaLength = errors.New("delta length is invalid")
