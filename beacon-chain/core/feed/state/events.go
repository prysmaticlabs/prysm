package state

import "time"

const (
	// BlockProcessed is sent after a block has been processed and updated the state database.
	BlockProcessed = iota + 1
	// ChainStarted is sent when enough validators are active to start proposing blocks.
	ChainStarted
	// StateInitialized is sent when the internal beacon node's state is ready to be accessed.
	StateInitialized
)

// BlockProcessedData is the data sent with BlockProcessed events.
type BlockProcessedData struct {
	// BlockHash is the hash of the processed block.
	BlockRoot [32]byte
	// Verified is true if the block's BLS contents have been verified.
	Verified bool
}

// ChainStartedData is the data sent with ChainStarted events.
type ChainStartedData struct {
	// StartTime is the time at which the chain started.
	StartTime time.Time
}

// StateInitializedData is the data sent with StateInitialized events.
type StateInitializedData struct {
	// StartTime is the time at which the chain started.
	StartTime time.Time
}
