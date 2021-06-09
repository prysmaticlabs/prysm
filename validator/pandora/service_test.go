package pandora

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestStart_Success method checks that service starts successfully or not
func TestStart_Success(t *testing.T) {
	hook := logTest.NewGlobal()
	pandoraService, err := NewService(context.Background(), HttpEndpoint, DialInProcRPCClient)
	require.NoError(t, err, "Error in preparing pandora mock service")

	pandoraService.Start()
	time.Sleep(1 * time.Second)

	if len(hook.Entries) > 0 {
		var want [2]string
		want[0] = "Could not connect to pandora chain"
		want[1] = "Could not check sync status of pandora chain"
		for _, entry := range hook.Entries {
			msg := entry.Message
			if strings.Contains(want[0], msg) {
				t.Errorf("incorrect log, expected %s, got %s", want[0], msg)
			}
			if strings.Contains(want[1], msg) {
				t.Errorf("incorrect log, expected %s, got %s", want[1], msg)
			}
		}
	}
	hook.Reset()
	pandoraService.cancel()
}

// Test_NoEndpointDefinedFails method checks invalid pandora chain endpoint
func Test_NoEndpointDefinedFails(t *testing.T) {
	_, err := NewService(context.Background(), "", DialRPCClient)
	want := "Pandora service initialization failed!"
	require.ErrorContains(t, want, err, "Should not initialize pandora service with empty endpoint")
}

// TestStop_OK method checks connection with pandora chain
func Test_WaitForConnection_ConnErr(t *testing.T) {
	pandoraService, err := NewService(context.Background(), HttpEndpoint, DialInProcRPCClient)
	require.NoError(t, err, "Error in preparing pandora mock service")

	status, err := pandoraService.isPandoraNodeSynced()
	require.NoError(t, err)
	require.Equal(t, true, status, "Should connect to pandora chain")
}

// TestStop_Success method checks service stop functionality
func TestStop_Success(t *testing.T) {
	pandoraService, err := NewService(context.Background(), HttpEndpoint, DialInProcRPCClient)
	require.NoError(t, err, "Error in preparing pandora mock service")
	err = pandoraService.Stop()
	require.NoError(t, err, "Unable to stop pandora chain service")
}

// TestService_GetShardBlockHeader_Success method checks GetWork method and test extraData decoding
func TestService_GetShardBlockHeader_Success(t *testing.T) {
	pandoraService, err := NewService(context.Background(), HttpEndpoint, DialInProcRPCClient)
	require.NoError(t, err, "Should not get error when preparing pandora mock service")
	pandoraService.connected = true
	pandoraService.isRunning = true

	actualHeader, actualHash, actualExtraData, err := pandoraService.GetShardBlockHeader(context.Background(), types.EmptyRootHash, 1000)
	require.NoError(t, err, "Should not get error when calling GetWork method")

	expectedExtraData, _, err := getDummyEncodedExtraData()
	require.NoError(t, err, "Should not get error when preparing encoded extraData")
	expectedBlock := getDummyBlock()
	if !reflect.DeepEqual(actualHeader, expectedBlock.Header()) {
		t.Errorf("incorrect block header %#v", actualHeader)
	}
	if !reflect.DeepEqual(actualHash, expectedBlock.Hash()) {
		t.Errorf("incorrect block hash %#v", actualHash)
	}
	if !reflect.DeepEqual(actualExtraData, expectedExtraData) {
		t.Errorf("incorrect extra data %#v", actualExtraData)
	}
}
