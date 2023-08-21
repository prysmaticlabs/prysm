package validator

import (
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

type ValidatorStatus int8

const (
	PendingInitialized ValidatorStatus = iota
	PendingQueued
	ActiveOngoing
	ActiveExiting
	ActiveSlashed
	ExitedUnslashed
	ExitedSlashed
	WithdrawalPossible
	WithdrawalDone
	Active
	Pending
	Exited
	Withdrawal
)

type SyncCommitteeSubscription struct {
	ValidatorIndex       primitives.ValidatorIndex
	SyncCommitteeIndices []uint64
	UntilEpoch           primitives.Epoch
}

type BeaconCommitteeSubscription struct {
	ValidatorIndex   primitives.ValidatorIndex
	CommitteeIndex   primitives.CommitteeIndex
	CommitteesAtSlot uint64
	Slot             primitives.Slot
	IsAggregator     bool
}
