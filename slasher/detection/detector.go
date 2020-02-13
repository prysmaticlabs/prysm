package detection

import (
	"context"

	"github.com/prysmaticlabs/prysm/slasher/db/kv"
)

// SlashingDetector defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type SlashingDetector struct {
	SlasherDB *kv.Store
	Ctx       context.Context
}
