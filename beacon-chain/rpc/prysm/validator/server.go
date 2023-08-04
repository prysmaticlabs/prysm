package validator

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/sync"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// Server defines a server implementation for HTTP endpoints, providing
// access data relevant to the Ethereum Beacon Chain.
type Server struct {
	GenesisTimeFetcher blockchain.TimeFetcher
	SyncChecker        sync.Checker
	HeadFetcher        blockchain.HeadFetcher
	SyncCommitteePool  synccommittee.Pool
	V1Alpha1Server     eth.BeaconNodeValidatorServer
}
