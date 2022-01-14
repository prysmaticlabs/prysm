package state

import ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"

type BeaconStateBellatrix interface {
	BeaconStateAltair
	LatestExecutionPayloadHeader() (*ethpb.ExecutionPayloadHeader, error)
	SetLatestExecutionPayloadHeader(payload *ethpb.ExecutionPayloadHeader) error
}
