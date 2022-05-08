package doublylinkedtree

import "errors"

var ErrNilNode = errors.New("invalid nil or unknown node")
var errInvalidParentRoot = errors.New("invalid parent root")
var errInvalidProposerBoostRoot = errors.New("invalid proposer boost root")
var errUnknownFinalizedRoot = errors.New("unknown finalized root")
var errUnknownJustifiedRoot = errors.New("unknown justified root")
var errInvalidOptimisticStatus = errors.New("invalid optimistic status")
var errUnknownPayloadHash = errors.New("unknown payload hash")
var errInvalidUnrealizedJustifiedEpoch = errors.New("invalid unrealized justified epoch")
var errInvalidUnrealizedFinalizedEpoch = errors.New("invalid unrealized finalized epoch")
