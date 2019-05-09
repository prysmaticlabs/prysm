package forkutil

import (
	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"

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
//
//
//  def get_domain(state: BeaconState,
//               domain_type: int,
//               message_epoch: int=None) -> int:
//    """
//    Return the signature domain (fork version concatenated with domain type) of a message.
//    """
//    epoch = get_current_epoch(state) if message_epoch is None else message_epoch
//    fork_version = state.fork.previous_version if epoch < state.fork.epoch else state.fork.current_version
//    return bytes_to_int(fork_version + int_to_bytes(domain_type, length=4))
func DomainVersion(state *pb.BeaconState, domainType uint64, epoch uint64) uint64 {
	if epoch == 0 {
		epoch = helpers.CurrentEpoch(state)
	}
	var forkVersion []byte
	if epoch < state.Fork.Epoch {
		forkVersion = state.Fork.PreviousVersion
	} else {
		forkVersion = state.Fork.CurrentVersion
	}
	by := []byte{}
	by = append(by, forkVersion...)
	by = append(by, bytesutil.Bytes4(domainType)...)
	return bytesutil.FromBytes8(by)
}
