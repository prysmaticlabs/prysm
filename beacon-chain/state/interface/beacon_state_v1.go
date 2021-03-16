package iface

import pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

// BeaconStateV1 has read and write access to beacon state V1/HF1 methods.
type BeaconStateV1 interface {
	BeaconState
	InnerStateUnsafe() *pbp2p.BeaconStateV1
}
