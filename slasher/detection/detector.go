package detection

import (
	"context"

	"github.com/prysmaticlabs/prysm/slasher/db"
)

// SlashingDetector defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type SlashingDetector struct {
	SlasherDB *db.Store
	Ctx       context.Context
}
