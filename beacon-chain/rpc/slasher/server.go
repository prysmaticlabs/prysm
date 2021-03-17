// Package slasher defines a gRPC server implementation of a slasher service
// which allows for checking if attestations or blocks are slashable.
package slasher

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/slasher/iface"
)

// Server defines a server implementation of the gRPC slasher service.
type Server struct {
	slasher iface.SlashableChecker
}
