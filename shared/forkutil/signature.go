package forkutil

import (
	"math"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ForkVersion returns the fork version of the given epoch number.
//
// Spec pseudocode definition:
//	def get_fork_version(fork: Fork,
//                     epoch: Epoch) -> int:
//    """
//    Return the fork version of the given ``epoch``.
//    """
//    if epoch < fork.epoch:
//        return fork.previous_version
//    else:
//        return fork.current_version
func ForkVersion(fork *pb.Fork, epoch uint64) uint64 {
	if epoch < fork.Epoch {
		return fork.PreviousVersion
	}
	return fork.CurrentVersion
}

// DomainVersion returns the domain version for BLS private key to sign and verify.
//
// Spec pseudocode definition:
//	def get_domain(fork: Fork,
//               epoch: Epoch,
//               domain_type: int) -> int:
//    """
//    Get the domain number that represents the fork meta and signature domain.
//    """
//    fork_version = get_fork_version(fork, epoch)
//    return fork_version * 2**32 + domain_type
func DomainVersion(fork *pb.Fork, epoch uint64, domainType uint64) uint64 {
	offset := uint64(math.Pow(2, 32))
	return ForkVersion(fork, epoch)*offset + domainType
}
