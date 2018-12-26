package misc

import (
	"math"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ForkVersion Spec:
//	def get_fork_version(fork_data: ForkData,
//                     slot: int) -> int:
//    if slot < fork_data.fork_slot:
//        return fork_data.pre_fork_version
//    else:
//        return fork_data.post_fork_version
func ForkVersion(data *pb.ForkData, slot uint64) uint64 {
	if slot < data.ForkSlot {
		return data.PreForkVersion
	}
	return data.PostForkVersion
}

// DomainVersion Spec:
//	def get_domain(fork_data: ForkData,
//               slot: int,
//               domain_type: int) -> int:
//    return get_fork_version(
//        fork_data,
//        slot
//    ) * 2**32 + domain_type
func DomainVersion(data *pb.ForkData, slot uint64, domainType uint64) uint64 {
	constant := uint64(math.Pow(2, 32))
	return ForkVersion(data, slot)*constant + domainType
}
