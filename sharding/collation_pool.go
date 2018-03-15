package sharding

import (
	"errors"
)

var (
	// ErrInvalidProposer is returned if the collation contains an invalid signature.
	ErrInvalidProposer = errors.New("invalid proposer")

	// ErrCollationNumTooLow is returned if the number of a collation is lower than the
	// one present in the shard chain.
	ErrCollationNumTooLow = errors.New("collation number too low")

	// ErrUnderpriced is returned if a proposer's bid price is below the minimum
	// configured for the proposer pool.
	ErrUnderpriced = errors.New("bid underpriced")

	// ErrReplaceUnderpriced is returned if a collation is attempted to be replaced
	// with a different one without the required price bump.
	ErrReplaceUnderpriced = errors.New("replacement collation underpriced")

	// ErrInsufficientFunds is returned if the proposed bid
	// is higher than the balance of the user's account.
	ErrInsufficientFunds = errors.New("insufficient funds for bid")

	// ErrIncorrectShardId is returned if the collation id
	// does not match current shard id.
	ErrIncorrectShardId = errors.New("incorrect shard id")

	//ErrExpectedPeriodNumber is returned if the expected perid number
	//has already passed.
	ErrExpectedPeriodNumber = errors.New("expected period number has passed")
)