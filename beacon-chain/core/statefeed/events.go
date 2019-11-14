package statefeed

import "time"

// EventType is the type that defines the type of event.
type EventType int

const (
	// BlockProcessed is sent after a block has been processed and updated the state database.
	// It is commonly used to provide
	BlockProcessed = iota + 1
	// ChainStarted is sent when enough validators are active to start proposing blocks.
	ChainStarted
)

// Event is the event that is sent with state feed updates
type Event struct {
	// Type is the type of event
	Type EventType
	// Data is event-specific data
	Data interface{}
}

// BlockReceivedData is the data sent with BlockReceived events.
type BlockReceivedData struct {
	// BlockHash is the hash of the received block.
	BlockHash [32]byte
}

// BlockProcessedData is the data sent with BlockProcessed events.
type BlockProcessedData struct {
	// BlockHash is the hash of the processed block.
	BlockHash [32]byte
}

// ChainStartedData is the data sent with ChainStarted events.
type ChainStartedData struct {
	// StartTime is the time at which the chain started.
	StartTime time.Time
}
