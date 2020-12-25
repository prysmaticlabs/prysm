package blocks

import (
	"bytes"
	"context"
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// def process_shard_header(state: BeaconState,
//                         signed_header: Signed[ShardDataHeader]) -> None:
//    header = signed_header.message
//    header_root = hash_tree_root(header)
//    # Verify signature
//    signer_index = get_shard_proposer_index(state, header.slot, header.shard)
//    assert bls.Verify(
//        state.validators[signer_index].pubkey,
//        compute_signing_root(header, get_domain(state, DOMAIN_SHARD_HEADER)),
//        signed_header.signature
//    )
//    # Verify length of the header, and simultaneously verify degree.
//    assert (
//        bls.Pairing(header.length_proof, SIZE_CHECK_POINTS[header.commitment.length]) ==
//        bls.Pairing(header.commitment.point, G2_SETUP[-header.commitment.length]))
//    )
//    # Get the correct pending header list
//    if compute_epoch_at_slot(header.slot) == get_current_epoch(state):
//        pending_headers = state.current_epoch_pending_shard_headers
//    else:
//        pending_headers = state.previous_epoch_pending_shard_headers
//
//    # Check that this header is not yet in the pending list
//    for pending_header in pending_headers:
//        assert header_root != pending_header.root
//    # Include it in the pending list
//    committee_length = len(get_beacon_committee(state, header.slot, header.shard))
//    pending_headers.append(PendingShardHeader(
//        slot=header.slot,
//        shard=header.shard,
//        commitment=header.commitment,
//        root=header_root,
//        votes=Bitlist[MAX_COMMITTEE_SIZE]([0] * committee_length),
//        confirmed=False
//    ))
func ProcessShardHeader(
	ctx context.Context,
	beaconState *stateTrie.BeaconState,
	header *ethpb.ShardHeader,
) (*stateTrie.BeaconState, error) {

	// Check header signature
	// Check header length proof against commitment length
	// Check header commitment point against commitment length

	var pendingHeaders []*pb.PendingShardHeader
	var currentEpoch bool
	if helpers.SlotToEpoch(header.Slot) == helpers.CurrentEpoch(beaconState) {
		pendingHeaders = beaconState.CurrentEpochPendingShardHeaders()
		currentEpoch = true
	} else {
		pendingHeaders = beaconState.PreviousEpochPendingShardHeaders()
		currentEpoch = false
	}

	// Check header is not yet in the pending list
	headerRoot, err := header.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	for _, pendingHeader := range pendingHeaders {
		if bytes.Equal(pendingHeader.Root, headerRoot[:]) {
			return nil, errors.New("incorrect header root")
		}
	}
	c, err := helpers.BeaconCommitteeFromState(beaconState, header.Slot, header.Shard) // TODO: Should be committee index
	if err != nil {
		return nil, err
	}

	if currentEpoch {
		beaconState.AppendCurrentEpochPendingShardHeader(&pb.PendingShardHeader{
			Slot:       header.Slot,
			Shard:      header.Shard,
			Commitment: header.Commitment.Point,
			Root:       headerRoot[:],
			Votes:      bitfield.NewBitlist(uint64(len(c))),
			Confirmed:  false,
		})
	} else {
		beaconState.AppendPreviousEpochPendingShardHeader(&pb.PendingShardHeader{
			Slot:       header.Slot,
			Shard:      header.Shard,
			Commitment: header.Commitment.Point,
			Root:       headerRoot[:],
			Votes:      bitfield.NewBitlist(uint64(len(c))),
			Confirmed:  false,
		})
	}

	return beaconState, nil
}
