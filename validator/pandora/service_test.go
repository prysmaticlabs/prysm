package pandora

import (
	"context"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"strings"
	"testing"
	"time"
)

// MockPandoraService method returns a mocked pandora service using
// mocked rpc client and mocked rpc server.
func MockPandoraService(endpoint string, dialPandoraFn DialRPCFn) (*Service, error) {
	mockedPandoraService, err := NewService(context.Background(), endpoint, dialPandoraFn)
	if err != nil {
		return nil, err
	}
	return mockedPandoraService, nil
}

// TestStart_OK method checks that service starts successfully or not
func TestStart_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	pandoraService, err := MockPandoraService(HttpEndpoint, DialInProcRPCClient)
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

func Test_NoEndpointDefinedFails(t *testing.T) {
	_, err := MockPandoraService("", DialRPCClient)
	want := "Pandora service initialization failed!"
	require.ErrorContains(t, want, err, "Should not initialize pandora service with empty endpoint")
}

func Test_WaitForConnection_ConnErr(t *testing.T) {
	pandoraService, err := MockPandoraService(HttpEndpoint, DialInProcRPCClient)
	require.NoError(t, err, "Error in preparing pandora mock service")

	status, _ := pandoraService.isPandoraNodeSynced()
	require.Equal(t, true, status, "Should connect to pandora chain")
}

func TestStop_OK(t *testing.T) {
	pandoraService, err := MockPandoraService(HttpEndpoint, DialInProcRPCClient)
	require.NoError(t, err, "Error in preparing pandora mock service")
	err = pandoraService.Stop()
	require.NoError(t, err, "Unable to stop pandora chain service")
}