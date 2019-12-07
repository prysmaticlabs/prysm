package operationFeed

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// How to add a new event to the feed:
//   1. Add a constant describing the event to the list below.
//   2. Add a structure with the name `<event>Data` containing any data fields that should be supplied with the event.
//
// Note that the same event is supplied to all subscribers, so the event received by subscribers should be considered read-only.

// EventType is the type that defines the type of event.
type EventType int

const (
	// UnaggregatedAttReceived is sent after an unaggregated attestation object has been received
	// from the outside world. (eg. in RPC or sync)
	UnaggregatedAttReceived = iota + 1

	// AggregatedAttReceived is sent after an aggregated attestation object has been received
	// from the outside world. (eg. in sync)
	AggregatedAttReceived = iota + 1

	// ExitReceived is sent after an voluntary exit object has been received from the outside world (eg in RPC or sync)
	ExitReceived
)

// Event is the event that is sent with operation feed updates.
type Event struct {
	// Type is the type of event.
	Type EventType
	// Data is event-specific data.
	Data interface{}
}

// UnAggregatedAttReceivedData is the data sent with UnaggregatedAttReceived events.
type UnAggregatedAttReceivedData struct {
	// Attestation is the unaggregated attestation object.
	Attestation *ethpb.Attestation
}

// AggregatedAttReceivedData is the data sent with AggregatedAttReceived events.
type AggregatedAttReceivedData struct {
	// Attestation is the aggregated attestation object.
	Attestation *pb.AggregateAndProof
}

// ExitRecievedData is the data sent with ExitReceived events.
type ExitRecievedData struct {
	// Exit is the voluntary exit object.
	Exit *ethpb.VoluntaryExit
}
