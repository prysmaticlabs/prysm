package validator

import "github.com/prysmaticlabs/prysm/beacon-chain/blockchain"

// Server defines a server implementation of the gRPC Validator service,
// providing RPC endpoints intended for validator clients.
type Server struct {
	HeadFetcher blockchain.HeadFetcher
	TimeFetcher blockchain.TimeFetcher
}
