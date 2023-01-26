package sync

import (
	"context"
	"testing"
	"time"

	mockChain "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/blstoexec"
	mockp2p "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func TestBroadcastBLSChanges(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig()
	c.CapellaForkEpoch = c.BellatrixForkEpoch.Add(2)
	params.OverrideBeaconConfig(c)
	chainService := &mockChain.ChainService{
		Genesis:        time.Now(),
		ValidatorsRoot: [32]byte{'A'},
	}
	s := NewService(context.Background(),
		WithP2P(mockp2p.NewTestP2P(t)),
		WithInitialSync(&mockSync.Sync{IsSyncing: false}),
		WithChainService(chainService),
		WithStateNotifier(chainService.StateNotifier()),
		WithOperationNotifier(chainService.OperationNotifier()),
		WithBlsToExecPool(blstoexec.NewPool()),
	)
	var emptySig [96]byte
	s.cfg.blsToExecPool.InsertBLSToExecChange(&ethpb.SignedBLSToExecutionChange{
		Message: &ethpb.BLSToExecutionChange{
			ValidatorIndex:     10,
			FromBlsPubkey:      make([]byte, 48),
			ToExecutionAddress: make([]byte, 20),
		},
		Signature: emptySig[:],
	})

	capellaStart, err := slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
	require.NoError(t, err)
	s.broadcastBLSChanges(capellaStart + 1)
}
