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
	root := bytesutil.ToBytes32(update.Header.StateRoot)
	leaf, err := update.NextSyncCommittee.HashTreeRoot()
	if err != nil {
		log.Fatalf("could not hash tree root: %v", err)
	}
	log.Infof("Verifying proof with root %#x, leaf %#x", root, leaf)
	validProof := ssz.VerifyProof(root, update.NextSyncCommitteeBranch, leaf, NextSyncCommitteeIndex)
	if !validProof {
		log.Error("could not verify merkle proof")
	}
}
