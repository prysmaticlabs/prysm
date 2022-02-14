package protoarray

import "errors"

var errNilNode = errors.New("invalid nil or unknown node")
var errInvalidBalance = errors.New("invalid node balance")
var errInvalidProposerBoostRoot = errors.New("invalid proposer boost root")
var errUnknownFinalizedRoot = errors.New("unknown finalized root")
var errUnknownJustifiedRoot = errors.New("unknown justified root")
var errInvalidOptimisticStatus = errors.New("invalid optimistic status")
