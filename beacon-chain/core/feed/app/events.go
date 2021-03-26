package app

import ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

const (
	// ProcessAppPayload is used to signal subscriber to process eth1 application payload data over the feed.
	ProcessAppPayload = iota + 1
	// ProduceAppPayload is used to signal subscribe to produce and return eth1 application payload data over the feed.
	// Typically the application payload data will be included inside a beacon block.
	ProduceAppPayload
)

type ProcessAppPayloadData struct {
	// TODO: Add beacon state's application block hash
	RandaoMix  []byte
	Slot       uint64
	AppPayload *ethpb.ApplicationPayload
}

type ProduceAppPayloadData struct {
	// TODO: Add beacon state's application block hash
	RandaoMix []byte
	Slot      uint64
}
