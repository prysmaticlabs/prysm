package validator

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	v1alpha1validator "github.com/prysmaticlabs/prysm/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
)

// Server defines a server implementation of the gRPC Validator service,
// providing RPC endpoints intended for validator clients.
type Server struct {
	HeadFetcher      blockchain.HeadFetcher
	TimeFetcher      blockchain.TimeFetcher
	SyncChecker      sync.Checker
	AttestationsPool attestations.Pool
	P2P              p2p.Broadcaster
	V1Alpha1Server   *v1alpha1validator.Server
}
