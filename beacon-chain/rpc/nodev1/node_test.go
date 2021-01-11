package nodev1

import (
	"context"
	"runtime"
	"strings"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	syncmock "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/version"
)

func TestGetVersion(t *testing.T) {
	semVer := version.GetSemanticVersion()
	os := runtime.GOOS
	arch := runtime.GOARCH
	res, err := (&Server{}).GetVersion(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	v := res.Data.Version
	assert.Equal(t, true, strings.Contains(v, semVer))
	assert.Equal(t, true, strings.Contains(v, os))
	assert.Equal(t, true, strings.Contains(v, arch))
}

func TestGetHealth(t *testing.T) {
	ctx := context.Background()
	checker := &syncmock.Sync{}
	s := &Server{
		SyncChecker: checker,
	}

	_, err := s.GetHealth(ctx, &ptypes.Empty{})
	require.ErrorContains(t, "node not initialized or having issues", err)
	checker.IsInitialized = true
	_, err = s.GetHealth(ctx, &ptypes.Empty{})
	require.NoError(t, err)
	checker.IsInitialized = false
	checker.IsSyncing = true
	require.NoError(t, err)
}

func TestSyncStatus(t *testing.T) {
	checker := &syncmock.Sync{Slot: 100}
	currentSlot := new(uint64)
	*currentSlot = 110
	timeFetcher := &mock.ChainService{Slot: currentSlot}

	s := &Server{
		SyncChecker:        checker,
		GenesisTimeFetcher: timeFetcher,
	}
	resp, err := s.GetSyncStatus(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, uint64(100), resp.Data.HeadSlot)
	assert.Equal(t, uint64(10), resp.Data.SyncDistance)
}
