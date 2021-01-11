package nodev1

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/emptypb"

	syncmock "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/version"
)

func TestGetVersion(t *testing.T) {
	semVer := version.GetSemanticVersion()
	os := runtime.GOOS
	arch := runtime.GOARCH
	res, err := (&Server{}).GetVersion(context.Background(), &emptypb.Empty{})
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

	_, err := s.GetHealth(ctx, &emptypb.Empty{})
	require.ErrorContains(t, "node not initialized or having issues", err)
	checker.IsInitialized = true
	_, err = s.GetHealth(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	checker.IsInitialized = false
	checker.IsSyncing = true
	require.NoError(t, err)
}
