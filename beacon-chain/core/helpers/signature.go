package helpers

import (
	"fmt"
	"math"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

// ForkVersion returns the fork version of the given epoch number.
//
// Spec pseudocode definition:
//	def get_fork_version(fork: Fork,
//                     epoch: EpochNumber) -> int:
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
//               epoch: EpochNumber,
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

// VerifyBitfield validates a bitfield with a given committee size.
//
// Spec pseudocode:
//
// def verify_bitfield(bitfield: bytes, committee_size: int) -> bool:
// """
// Verify ``bitfield`` against the ``committee_size``.
// """
// if len(bitfield) != (committee_size + 7) // 8:
// return False
//
// # Check `bitfield` is padded with zero bits only
// for i in range(committee_size, len(bitfield) * 8):
// if get_bitfield_bit(bitfield, i) == 0b1:
// return False
//
// return True
func VerifyBitfield(bitfield []byte, committee_size int) (bool, error) {
	if len(bitfield) != mathutil.CeilDiv8(committee_size) {
		return false, fmt.Errorf(
			"wanted participants bitfield length %d, got: %d",
			mathutil.CeilDiv8(committee_size),
			len(bitfield))
	}

	for i := committee_size; i < len(bitfield); i++ {
		bitSet, err := bitutil.CheckBit(bitfield, i)
		if err != nil {
			return false, fmt.Errorf("unable to check bit in bitfield %v", err)
		}

		if !bitSet {
			return false, nil
		}
	}

	return true, nil
}
