package main

import (
	"context"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Precomputed values for generalized indices.
const (
	FinalizedRootIndex              = 105
	FinalizedRootIndexFloorLog2     = 6
	NextSyncCommitteeIndex          = 55
	NextSyncCommitteeIndexFloorLog2 = 5
)

var log = logrus.WithField("prefix", "light")

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

func main() {
	conn, err := grpc.Dial("localhost:4000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	ctx := context.Background()
	lesClient := v1alpha1.NewLightClientClient(conn)
	update, err := lesClient.LatestUpdateFinalized(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("could not get latest update: %v", err)
	}
	// Attempt to verify a merkle proof of the next sync committee branch vs. the state root.
	root := bytesutil.ToBytes32(update.FinalityHeader.StateRoot)
	nextSyncCommitteeRoot, err := update.NextSyncCommittee.HashTreeRoot()
	if err != nil {
		log.Fatalf("could not hash tree root: %v", err)
	}
	validProof := ssz.VerifyProof(root, update.NextSyncCommitteeBranch, nextSyncCommitteeRoot, NextSyncCommitteeIndex)
	if !validProof {
		log.Error("could not verify merkle proof")
	}
	//
	//// Get basic information such as the genesis validators root.
	//genesis, err := beaconClient.GetGenesis(ctx, &emptypb.Empty{})
	//if err != nil {
	//	panic(err)
	//}
	//genesisValidatorsRoot := genesis.Data.GenesisValidatorsRoot
	//genesisTime := uint64(genesis.Data.GenesisTime.AsTime().Unix())
	//fmt.Printf("%#v\n", genesisValidatorsRoot)
	//currentState, err := debugClient.GetBeaconStateV2(ctx, &v2.StateRequestV2{StateId: []byte("head")})
	//if err != nil {
	//	panic(err)
	//}
	//altairState := currentState.Data.GetAltairState()
	//store := &Store{
	//	Snapshot: &LightClientSnapshot{
	//		Header:               nil,
	//		CurrentSyncCommittee: altairState.CurrentSyncCommittee,
	//		NextSyncCommittee:    altairState.NextSyncCommittee,
	//	},
	//	ValidUpdates: make([]*LightClientUpdate, 0),
	//}
	//
	//events, err := eventsClient.StreamEvents(ctx, &v1.StreamEventsRequest{Topics: []string{"head"}})
	//if err != nil {
	//	panic(err)
	//}
	//for {
	//	item, err := events.Recv()
	//	if err != nil {
	//		panic(err)
	//	}
	//	evHeader := &v1.EventHead{}
	//	if err := item.Data.UnmarshalTo(evHeader); err != nil {
	//		panic(err)
	//	}
	//	blockHeader, err := beaconClient.GetBlockHeader(ctx, &v1.BlockRequest{BlockId: evHeader.Block})
	//	if err != nil {
	//		panic(err)
	//	}
	//	store.Snapshot.Header = blockHeader.Data.Header.Message
	//	fmt.Println(store)
	//	currentSlot := slots.CurrentSlot(genesisTime)
	//	if err := processLightClientUpdate(
	//		store,
	//		&LightClientUpdate{},
	//		currentSlot,
	//		bytesutil.ToBytes32(genesisValidatorsRoot),
	//	); err != nil {
	//		panic(err)
	//	}
	//}
}
