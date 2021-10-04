package main

import (
	"context"
	"fmt"
	"log"

	eth "github.com/prysmaticlabs/prysm/proto/eth/service"
	v1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	"google.golang.org/grpc"
)

type LightClientSnapshot struct {
	Header               *v1.BeaconBlockHeader
	CurrentSyncCommittee *v2.SyncCommittee
	NextSyncCommittee    *v2.SyncCommittee
}

func main() {
	conn, err := grpc.Dial("localhost:4000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	beaconEndpoint := eth.NewBeaconChainClient(conn)
	blk, err := beaconEndpoint.GetBlockHeader(context.Background(), &v1.BlockRequest{
		BlockId: []byte("finalized"),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(blk)
}

func validateLightClietUpdate(snapshot *LightClientSnapshot) error {
	return nil
}
