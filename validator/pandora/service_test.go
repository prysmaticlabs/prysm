package pandora

import (
	"context"
	"github.com/ethereum/go-ethereum/rpc"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"strings"
	"testing"
)

var httpEndpoint = "http://127.0.0.1:4045"

func mockDialRPC(endpoint string) (*PandoraClient, *rpc.Client, error) {
	log.Info("in mockDialRPC method")
	server := NewMockPandoraServer()
	rpcClient := rpc.DialInProc(server)
	pandoraClient := NewClient(rpcClient)

	return pandoraClient, rpcClient, nil
}

func TestStart_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	pandoraService := NewService(
		context.Background(),
		httpEndpoint, mockDialRPC)

	pandoraService.Start()
	//time.Sleep(2 * time.Second)
	if len(hook.Entries) > 0 {
		msg := hook.LastEntry().Message
		want := "Could not connect to ETH1.0 chain RPC client"
		if strings.Contains(want, msg) {
			t.Errorf("incorrect log, expected %s, got %s", want, msg)
		}
	}
	hook.Reset()
	pandoraService.cancel()
}