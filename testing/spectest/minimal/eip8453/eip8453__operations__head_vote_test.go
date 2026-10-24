package eip8453

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/testing/spectest/shared/eip8453/operations"
)

func TestMinimal_EIP8453_Operations_HeadVote(t *testing.T) {
	operations.RunHeadVoteTest(t, "minimal")
}
