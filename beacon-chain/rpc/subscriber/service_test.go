package subscriber

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/beacon"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"testing"
)

func setup() (*Config, error) {
	return &Config{
		IPCPath:    cmd.DefaultIpcPath,
		HTTPEnable: true,
		HTTPHost:   cmd.DefaultHTTPHost,
		HTTPPort:   cmd.DefaultHTTPPort,
		WSEnable:   true,
		WSHost:     cmd.DefaultWSHost,
		WSPort:     cmd.DefaultWSPort,
	}, nil
}

// TestServerStart_Success
func TestServerStart_Success(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	config, err := setup()
	assert.NoError(t, err)

	beaconServer := beacon.Server{}

	rpcService, err := NewService(ctx, config, beaconServer)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}

	// Ensure that a node can be successfully started, but only once
	assert.NoError(t, rpcService.startRPC())
	require.LogsContain(t, hook, "IPC endpoint opened", "IPC server not started")
	require.LogsContain(t, hook, "HTTP server started", "Http server not started")
	require.LogsContain(t, hook, "WebSocket enabled", "Web socket server not started")

	hook.Reset()
	assert.NoError(t, rpcService.Stop())
}
