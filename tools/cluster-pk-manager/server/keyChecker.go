package main

import (
	"context"

	pbBeacon "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
)

type keyChecker struct {
	db     *db
	client pbBeacon.ValidatorServiceClient
}

func newkeyChecker(db *db, beaconRPCAddr string) *keyChecker {
	// connect to the beacon node
	dialOpt := grpc.WithInsecure()
	conn, err := grpc.DialContext(context.Background(), beaconRPCAddr, dialOpt, grpc.WithStatsHandler(&ocgrpc.ClientHandler{}))
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", beaconRPCAddr, err)
	}
	valClient := pbBeacon.NewValidatorServiceClient(conn)

	return &keyChecker{
		db:     db,
		client: valClient,
	}
}

func (k *keyChecker) checkKeys() {
	pubkeys, keyMap, err := k.db.KeyMap()
	if err != nil {
		log.Error(err)
	}
}
