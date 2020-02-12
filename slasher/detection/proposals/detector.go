package attestations

import (
	"context"

	"github.com/prysmaticlabs/prysm/slasher/db"
)

// ProposerDetector defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type ProposerDetector struct {
	SlasherDB *db.Store
	ctx       context.Context
}
