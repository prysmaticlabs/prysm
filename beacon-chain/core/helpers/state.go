package helpers

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// StateRoot returns the state root stored in the BeaconState for a recent slot.
// It returns an error if the requested state root is not within the slot range.
// Spec pseudocode definition:
// 	def get_state_root(state: BeaconState,
//                   slot: Slot) -> Bytes32:
//    """
//    Return the state root at a recent ``slot``.
//    """
//    assert slot < state.slot <= slot + SLOTS_PER_HISTORICAL_ROOT
//    return state.latest_state_roots[slot % SLOTS_PER_HISTORICAL_ROOT]
func StateRoot(state *pb.BeaconState, slot uint64) ([]byte, error) {
	earliestSlot := state.Slot - params.BeaconConfig().SlotsPerHistoricalRoot

	if slot < earliestSlot || slot >= state.Slot {
		if earliestSlot < params.BeaconConfig().GenesisSlot {
			earliestSlot = params.BeaconConfig().GenesisSlot
		}
		return []byte{}, fmt.Errorf("slot %d is not within range %d to %d",
			slot-params.BeaconConfig().GenesisSlot,
			earliestSlot-params.BeaconConfig().GenesisSlot,
			state.Slot-params.BeaconConfig().GenesisSlot,
		)
	}
	return state.LatestStateRoots[slot%params.BeaconConfig().SlotsPerHistoricalRoot], nil
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
func DomainVersion(state *pb.BeaconState, epoch uint64, domainType uint64) uint64 {
	if epoch == 0 {
		epoch = CurrentEpoch(state)
	}
	var forkVersion []byte
	if epoch < state.Fork.Epoch {
		forkVersion = state.Fork.PreviousVersion
	} else {
		forkVersion = state.Fork.CurrentVersion
	}
	by := []byte{}
	by = append(by, forkVersion[:4]...)
	by = append(by, bytesutil.Bytes4(domainType)...)
	return bytesutil.FromBytes8(by)
}
