package validator

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/synccommittee"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	v1alpha1validator "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/statefetcher"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/sync"
)

// Server defines a server implementation of the gRPC Validator service,
// providing RPC endpoints intended for validator clients.
type Server struct {
	HeadFetcher           blockchain.HeadFetcher
	HeadUpdater           blockchain.HeadUpdater
	TimeFetcher           blockchain.TimeFetcher
	SyncChecker           sync.Checker
	AttestationsPool      attestations.Pool
	PeerManager           p2p.PeerManager
	Broadcaster           p2p.Broadcaster
	StateFetcher          statefetcher.Fetcher
	OptimisticModeFetcher blockchain.OptimisticModeFetcher
	SyncCommitteePool     synccommittee.Pool
	V1Alpha1Server        *v1alpha1validator.Server
}
