package types

import "github.com/pkg/errors"

// BroadcastValidation specifies the level of validation that must be applied to a block before it is broadcast.
type BroadcastValidation uint8

const (
	// Gossip represents lightweight gossip checks only.
	Gossip BroadcastValidation = iota
	// Consensus represents full consensus checks, including validation of all signatures and
	// blocks fields except for the execution payload transactions..
	Consensus
	// ConsensusAndEquivocation the same as Consensus, with an extra equivocation
	// check immediately before the block is broadcast. If the block is found to be an
	// equivocation it fails validation.
	ConsensusAndEquivocation
)

var (
	// ErrConsensusValidationFailed means that a block failed consensus checks.
	ErrConsensusValidationFailed = errors.New("block failed consensus validation")
	// ErrEquivocationValidationFailed means that a block failed equivocation checks.
	ErrEquivocationValidationFailed = errors.New("block failed equivocation validation")
)
