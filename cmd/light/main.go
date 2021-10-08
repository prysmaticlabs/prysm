package main

import (
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
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/proto/eth/service"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
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
	ForkVersion             *v1alpha1.Version
}

type Store struct {
	Snapshot     *LightClientSnapshot
	ValidUpdates []*LightClientUpdate
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
	store.ValidUpdates = append(store.ValidUpdates, update)
	updateTimeout := uint64(params.BeaconConfig().SlotsPerEpoch) * uint64(params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	sumParticipantBits := sumBits(update.SyncCommitteeBits)
	hasQuorum := sumParticipantBits*3 >= uint64(len(update.SyncCommitteeBits))*2
	if hasQuorum && !isEmptyBlockHeader(update.FinalityHeader) {
		// Apply update if (1) 2/3 quorum is reached and (2) we have a finality proof.
		// Note that (2) means that the current light client design needs finality.
		// It may be changed to re-organizable light client design. See the on-going issue consensus-specs#2182.
		applyLightClientUpdate(store.Snapshot, update)
		store.ValidUpdates = make([]*LightClientUpdate, 0)
	} else if currentSlot > store.Snapshot.Header.Slot.Add(updateTimeout) {
		// Forced best update when the update timeout has elapsed
		// Use the update that has the highest sum of sync committee bits.
		updateWithHighestSumBits := store.ValidUpdates[0]
		highestSumBitsUpdate := sumBits(updateWithHighestSumBits.SyncCommitteeBits)
		for _, validUpdate := range store.ValidUpdates {
			sumUpdateBits := sumBits(validUpdate.SyncCommitteeBits)
			if sumUpdateBits > highestSumBitsUpdate {
				highestSumBitsUpdate = sumUpdateBits
				updateWithHighestSumBits = validUpdate
			}
		}
		applyLightClientUpdate(store.Snapshot, updateWithHighestSumBits)
		store.ValidUpdates = make([]*LightClientUpdate, 0)
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

	// Verify finality headers.
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
		leaf, err := update.Header.HashTreeRoot()
		if err != nil {
			return err
		}
		depth := FinalizedRootIndexFloorLog2
		index := getSubtreeIndex(FinalizedRootIndex)
		root := update.FinalityHeader.StateRoot
		merkleBranch := make([][]byte, len(update.FinalityBranch))
		for i, item := range update.FinalityBranch {
			merkleBranch[i] = item[:]
		}
		if !trie.VerifyMerkleBranch(root, leaf[:], int(index), merkleBranch, uint64(depth)) {
			return errors.New("does not verify")
		}
	}

	// Verify update next sync committee if the update period incremented.
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
		v1Sync := &v1alpha1.SyncCommittee{
			Pubkeys:         syncCommittee.Pubkeys,
			AggregatePubkey: syncCommittee.AggregatePubkey,
		}
		leaf, err := v1Sync.HashTreeRoot()
		if err != nil {
			return err
		}
		depth := NextSyncCommitteeIndexFloorLog2
		index := getSubtreeIndex(NextSyncCommitteeIndex)
		root := update.Header.StateRoot
		merkleBranch := make([][]byte, len(update.NextSyncCommitteeBranch))
		for i, item := range update.NextSyncCommitteeBranch {
			merkleBranch[i] = item[:]
		}
		if !trie.VerifyMerkleBranch(root, leaf[:], int(index), merkleBranch, uint64(depth)) {
			return errors.New("does not verify")
		}
	}

	// Verify sync committee has sufficient participants
	if sumBits(update.SyncCommitteeBits) < params.BeaconConfig().MinSyncCommitteeParticipants {
		return errors.New("insufficient participants")
	}

	// Verify sync committee aggregate signature
	participantPubkeys := make([][]byte, 0)
	for i, pubKey := range syncCommittee.Pubkeys {
		bit := update.SyncCommitteeBits.BitAt(uint64(i))
		if bit {
			participantPubkeys = append(participantPubkeys, pubKey)
		}
	}
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
	for _, pubkey := range participantPubkeys {
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
	beaconClient := eth.NewBeaconChainClient(conn)
	eventsClient := eth.NewEventsClient(conn)
	debugClient := eth.NewBeaconDebugClient(conn)
	ctx := context.Background()

	// Get basic information such as the genesis validators root.
	genesis, err := beaconClient.GetGenesis(ctx, &emptypb.Empty{})
	if err != nil {
		panic(err)
	}
	genesisValidatorsRoot := genesis.Data.GenesisValidatorsRoot
	genesisTime := uint64(genesis.Data.GenesisTime.AsTime().Unix())
	fmt.Printf("%#v\n", genesisValidatorsRoot)
	currentState, err := debugClient.GetBeaconStateV2(ctx, &v2.StateRequestV2{StateId: []byte("head")})
	if err != nil {
		panic(err)
	}
	altairState := currentState.Data.GetAltairState()
	store := &Store{
		Snapshot: &LightClientSnapshot{
			Header:               nil,
			CurrentSyncCommittee: altairState.CurrentSyncCommittee,
			NextSyncCommittee:    altairState.NextSyncCommittee,
		},
		ValidUpdates: make([]*LightClientUpdate, 0),
	}

	events, err := eventsClient.StreamEvents(ctx, &v1.StreamEventsRequest{Topics: []string{"head"}})
	if err != nil {
		panic(err)
	}
	for {
		item, err := events.Recv()
		if err != nil {
			panic(err)
		}
		evHeader := &v1.EventHead{}
		if err := item.Data.UnmarshalTo(evHeader); err != nil {
			panic(err)
		}
		blockHeader, err := beaconClient.GetBlockHeader(ctx, &v1.BlockRequest{BlockId: evHeader.Block})
		if err != nil {
			panic(err)
		}
		store.Snapshot.Header = blockHeader.Data.Header.Message
		fmt.Println(store)
		currentSlot := slots.CurrentSlot(genesisTime)
		if err := processLightClientUpdate(
			store,
			&LightClientUpdate{},
			currentSlot,
			bytesutil.ToBytes32(genesisValidatorsRoot),
		); err != nil {
			panic(err)
		}
	}
}

func sumBits(bfield bitfield.Bitvector512) uint64 {
	return bfield.Count()
}

func getSubtreeIndex(index uint64) uint64 {
	return index % uint64(math.Pow(2, floorLog2(index)))
}

func floorLog2(x uint64) float64 {
	return math.Floor(math.Log2(float64(x)))
}
