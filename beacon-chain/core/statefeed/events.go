package statefeed

import "time"

// How to add a new event to the feed:
//   1. Add a constant describing the event to the list below
//   2. Add a structure with the name `<event>Data` containing any data fields that should be supplied with the event.
//
// Note that the same event is supplied to all subscribers, so the event received by subscribers should be considered read-only.

// EventType is the type that defines the type of event.
type EventType int

const (
	// BlockProcessed is sent after a block has been processed and updated the state database.
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
