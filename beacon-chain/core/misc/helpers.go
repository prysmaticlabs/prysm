package misc

import (
	"math"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func ForkVersion(data *pb.ForkData, slot uint64) uint64 {
	if slot < data.ForkSlot {
		return data.PreForkVersion
	}
	return data.PostForkVersion
}

func DomainVersion(data *pb.ForkData, slot uint64, domainType uint64) uint64 {
	constant := uint64(math.Pow(2, 32))
	return ForkVersion(data, slot)*constant + domainType
}
