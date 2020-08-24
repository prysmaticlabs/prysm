// Package state contains types for state operation-specific events fired
// during the runtime of a beacon node such state initialization, state updates,
// and chain start.
package state

import "time"

const (
	// BlockProcessed is sent after a block has been processed and updated the state database.
	BlockProcessed = iota + 1
	// ChainStarted is sent when enough validators are active to start proposing blocks.
	ChainStarted
	// Initialized is sent when the internal beacon node's state is ready to be accessed.
	Initialized
	// Synced is sent when the beacon node has completed syncing and is ready to participate in the network.
	Synced
	// Reorg is an event sent when the new head state's slot after a block
	// transition is lower than its previous head state slot value.
	Reorg
)

// BlockProcessedData is the data sent with BlockProcessed events.
type BlockProcessedData struct {
	// Slot is the slot of the processed block.
	Slot uint64
	// BlockRoot of the processed block.
	BlockRoot [32]byte
	// Verified is true if the block's BLS contents have been verified.
	Verified bool
}

// ChainStartedData is the data sent with ChainStarted events.
type ChainStartedData struct {
	// StartTime is the time at which the chain started.
	StartTime time.Time
}

// SyncedData is the data sent with Synced events.
type SyncedData struct {
	// StartTime is the time at which the chain started.
	StartTime time.Time
}

// InitializedData is the data sent with Initialized events.
type InitializedData struct {
	// StartTime is the time at which the chain started.
	StartTime time.Time
	// GenesisValidatorsRoot represents state.validators.HashTreeRoot().
	GenesisValidatorsRoot []byte
}

// ReorgData is the data alongside a reorg event.
type ReorgData struct {
	// NewSlot is the slot of new state after the reorg.
	NewSlot uint64
	// OldSlot is the slot of the head state before the reorg.
	OldSlot uint64
}
