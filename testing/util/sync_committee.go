package util

import (
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// HydrateSyncCommittee hydrates the provided sync committee message.
func HydrateSyncCommittee(s *ethpb.SyncCommitteeMessage) *ethpb.SyncCommitteeMessage {
	if s.Signature == nil {
		s.Signature = make([]byte, 96)
	}
	if s.BlockRoot == nil {
		s.BlockRoot = make([]byte, 32)
	}
	return s
}
