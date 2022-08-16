package protoarray

import "errors"

var errUnknownFinalizedRoot = errors.New("unknown finalized root")
var errUnknownJustifiedRoot = errors.New("unknown justified root")
var errInvalidNodeIndex = errors.New("node index is invalid")
var errInvalidFinalizedNode = errors.New("invalid finalized block on chain")
var ErrUnknownNodeRoot = errors.New("unknown block root")
var errInvalidJustifiedIndex = errors.New("justified index is invalid")
var errInvalidBestDescendantIndex = errors.New("best descendant index is invalid")
var errInvalidParentDelta = errors.New("parent delta is invalid")
var errInvalidNodeDelta = errors.New("node delta is invalid")
var errInvalidDeltaLength = errors.New("delta length is invalid")
var errInvalidOptimisticStatus = errors.New("invalid optimistic status")
var errInvalidNilCheckpoint = errors.New("invalid nil checkpoint")
var errInvalidUnrealizedJustifiedEpoch = errors.New("invalid unrealized justified epoch")
var errInvalidUnrealizedFinalizedEpoch = errors.New("invalid unrealized finalized epoch")
var errNilBlockHeader = errors.New("invalid nil block header")
