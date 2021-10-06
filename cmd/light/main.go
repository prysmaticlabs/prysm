package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/trie"
	"github.com/prysmaticlabs/prysm/crypto/bls/blst"
	"github.com/prysmaticlabs/prysm/crypto/bls/common"
	eth "github.com/prysmaticlabs/prysm/proto/eth/service"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// Precomputed values for generalized indices.
const (
	FinalizedRootIndex              = 105
	FinalizedRootIndexFloorLog2     = 6
	NextSyncCommitteeIndex          = 55
	NextSyncCommitteeIndexFloorLog2 = 5
)

type LightClientSnapshot struct {
	Header               *v1.BeaconBlockHeader
	CurrentSyncCommittee *v2.SyncCommittee
	NextSyncCommittee    *v2.SyncCommittee
}

type LightClientUpdate struct {
	Header                  *v1.BeaconBlockHeader
	NextSyncCommittee       *v2.SyncCommittee
	NextSyncCommitteeBranch [NextSyncCommitteeIndexFloorLog2][32]byte
	FinalityHeader          *v1.BeaconBlockHeader
	FinalityBranch          [FinalizedRootIndexFloorLog2][32]byte
	SyncCommitteeBits       bitfield.Bitvector512
	SyncCommitteeSignature  [96]byte
	ForkVersion             *v1.Version
}

type updateSet []*LightClientUpdate

func (u *updateSet) add(update *LightClientUpdate) {
	// TODO: Implement.
}

type Store struct {
	Snapshot     *LightClientSnapshot
	ValidUpdates *updateSet
}

func applyLightClientUpdate(snapshot *LightClientSnapshot, update *LightClientUpdate) {
	snapshotPeriod := slots.ToEpoch(snapshot.Header.Slot) / params.BeaconConfig().EpochsPerSyncCommitteePeriod
	updatePeriod := slots.ToEpoch(update.Header.Slot) / params.BeaconConfig().EpochsPerSyncCommitteePeriod
	if updatePeriod == snapshotPeriod+1 {
		snapshot.CurrentSyncCommittee = snapshot.NextSyncCommittee
	} else {
		snapshot.Header = update.Header
	}
}

func processLightClientUpdate(
	store *Store,
	update *LightClientUpdate,
	currentSlot types.Slot,
	genesisValidatorsRoot [32]byte,
) error {
	if err := validateLightClientUpdate(store.Snapshot, update, genesisValidatorsRoot); err != nil {
		return err
	}
	store.ValidUpdates.add(update)
	updateTimeout := uint64(params.BeaconConfig().SlotsPerEpoch) * uint64(params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	// TODO: Add bits instead:
	sumParticipantBits := uint64(sumBits(update.SyncCommitteeBits))
	hasQuorum := sumParticipantBits*3 >= uint64(len(update.SyncCommitteeBits))*2
	if hasQuorum && !isEmptyBlockHeader(update.FinalityHeader) {
		// Apply update if (1) 2/3 quorum is reached and (2) we have a finality proof.
		// Note that (2) means that the current light client design needs finality.
		// It may be changed to re-organizable light client design. See the on-going issue consensus-specs#2182.
		applyLightClientUpdate(store.Snapshot, update)
		store.ValidUpdates = &updateSet{}
	} else if currentSlot > store.Snapshot.Header.Slot.Add(updateTimeout) {
		// Forced best update when the update timeout has elapsed
		//apply_light_client_update(store.snapshot,
		//	max(store.valid_updates, key=lambda update: sum(update.sync_committee_bits)))
		store.ValidUpdates = &updateSet{}
	}
	return nil
}

func validateLightClientUpdate(
	snapshot *LightClientSnapshot,
	update *LightClientUpdate,
	genesisValidatorsRoot [32]byte,
) error {
	if update.Header.Slot <= snapshot.Header.Slot {
		return errors.New("wrong")
	}
	snapshotPeriod := slots.ToEpoch(snapshot.Header.Slot) / params.BeaconConfig().EpochsPerSyncCommitteePeriod
	updatePeriod := slots.ToEpoch(update.Header.Slot) / params.BeaconConfig().EpochsPerSyncCommitteePeriod
	if updatePeriod != snapshotPeriod || updatePeriod != snapshotPeriod+1 {
		return errors.New("unwanted")
	}
	var signedHeader *v1.BeaconBlockHeader
	if isEmptyBlockHeader(update.FinalityHeader) {
		signedHeader = update.Header
		// Check if branch is empty.
		for _, elem := range update.FinalityBranch {
			if elem != [32]byte{} {
				return errors.New("branch not empty")
			}
		}
	} else {

	}
	leaf, err := update.Header.HashTreeRoot()
	if err != nil {
		return err
	}
	depth := uint64(2)
	index := getSubtreeIndex(2)
	root := update.FinalityHeader.StateRoot
	branch := update.FinalityBranch
	_ = branch
	//	signed_header = update.finality_header
	//	assert is_valid_merkle_branch(
	//		leaf=hash_tree_root(update.header),
	//		branch=update.finality_branch,
	//		depth=floorlog2(FINALIZED_ROOT_INDEX),
	//		index=get_subtree_index(FINALIZED_ROOT_INDEX),
	//		root=update.finality_header.state_root,
	//)
	if !trie.VerifyMerkleBranch(root, leaf[:], int(index), [][]byte{}, depth) {
		return errors.New("does not verify")
	}

	var syncCommittee *v2.SyncCommittee
	if updatePeriod == snapshotPeriod {
		syncCommittee = snapshot.CurrentSyncCommittee
		for _, elem := range update.NextSyncCommitteeBranch {
			if elem != [32]byte{} {
				return errors.New("branch not empty")
			}
		}
	} else {
		syncCommittee = snapshot.NextSyncCommittee
	}
	//# Verify update next sync committee if the update period incremented
	//if update_period == snapshot_period:
	//sync_committee = snapshot.current_sync_committee
	//assert update.next_sync_committee_branch == [Bytes32() for _ in range(floorlog2(NEXT_SYNC_COMMITTEE_INDEX))]
	//else:
	//sync_committee = snapshot.next_sync_committee
	//assert is_valid_merkle_branch(
	//leaf=hash_tree_root(update.next_sync_committee),
	//branch=update.next_sync_committee_branch,
	//depth=floorlog2(NEXT_SYNC_COMMITTEE_INDEX),
	//index=get_subtree_index(NEXT_SYNC_COMMITTEE_INDEX),
	//root=update.header.state_root,
	//)
	//

	// Verify sync committee has sufficient participants
	if uint64(sumBits(update.SyncCommitteeBits)) < params.BeaconConfig().MinSyncCommitteeParticipants {
		return errors.New("insufficient participants")
	}
	//
	//# Verify sync committee aggregate signature
	//participant_pubkeys = [pubkey for (bit, pubkey) in zip(update.sync_committee_bits, sync_committee.pubkeys) if bit]
	domain, err := signing.ComputeDomain(
		params.BeaconConfig().DomainSyncCommittee,
		[]byte(update.ForkVersion.Version),
		genesisValidatorsRoot[:],
	)
	if err != nil {
		return err
	}
	signingRoot, err := signing.ComputeSigningRoot(signedHeader, domain)
	if err != nil {
		return err
	}
	sig, err := blst.SignatureFromBytes(update.SyncCommitteeSignature[:])
	if err != nil {
		return err
	}
	pubKeys := make([]common.PublicKey, 0)
	for _, pubkey := range snapshot.CurrentSyncCommittee.Pubkeys {
		pk, err := blst.PublicKeyFromBytes(pubkey)
		if err != nil {
			return err
		}
		pubKeys = append(pubKeys, pk)
	}
	if !sig.FastAggregateVerify(pubKeys, signingRoot) {
		return errors.New("failed to verify")
	}
	return nil
}

func isEmptyBlockHeader(header *v1.BeaconBlockHeader) bool {
	emptyRoot := params.BeaconConfig().ZeroHash
	return proto.Equal(header, &v1.BeaconBlockHeader{
		Slot:          0,
		ProposerIndex: 0,
		ParentRoot:    emptyRoot[:],
		StateRoot:     emptyRoot[:],
		BodyRoot:      emptyRoot[:],
	})
}

func main() {
	conn, err := grpc.Dial("localhost:4000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	evs := eth.NewEventsClient(conn)
	ctx := context.Background()
	events, err := evs.StreamEvents(ctx, &v1.StreamEventsRequest{Topics: []string{"head"}})
	if err != nil {
		panic(err)
	}
	for {
		item, err := events.Recv()
		if err != nil {
			panic(err)
		}
		fmt.Println(item)
	}
}

func sumBits(bfield bitfield.Bitvector512) uint8 {
	s := uint8(0)
	for _, item := range bfield.Bytes() {
		s += item
	}
	return s
}

func getSubtreeIndex(index uint64) uint64 {
	return index % uint64(math.Pow(2, floorLog2(index)))
}

func floorLog2(x uint64) float64 {
	return math.Floor(math.Log2(float64(x)))
}
