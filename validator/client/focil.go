package client

// Focil stands for Fork-Choice enforced Inclusion Lists (FOCIL): A simple committee-based inclusion list proposal
// mechanism that allows validators to propose and vote on inclusion lists for the fork choice rule.
type Focil interface {
	//GetInclusionLists(ctx context.Context, slot primitives.Slot) ([]*ethpb.InclusionList, error)
}
