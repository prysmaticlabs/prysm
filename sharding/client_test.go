package sharding

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/rpc"
	cli "gopkg.in/urfave/cli.v1"
)

func randomEndpoint() string {
	return fmt.Sprintf("/tmp/go-ethereum-test-ipc-%d-%d", os.Getpid(), rand.Int63())
}

func newTestServer(endpoint string) (*rpc.Server, error) {
	server := rpc.NewServer()

	l, err := rpc.CreateIPCListener(endpoint)
	if err != nil {
		return nil, err
	}
	go server.ServeListener(l)

	return server, nil
}

func createContext() *cli.Context {
	set := flag.NewFlagSet("test", 0)
	set.String(utils.DataDirFlag.Name, "", "")
	return cli.NewContext(nil, set, nil)
}

func TestStart(t *testing.T) {
	endpoint := randomEndpoint()
	server, err := newTestServer(endpoint)
	if err != nil {
		t.Fatalf("Failed to create a test server: %v", err)
	}
	defer server.Stop()

	ctx := createContext()
	if err := ctx.GlobalSet(utils.DataDirFlag.Name, endpoint); err != nil {
		t.Fatalf("Failed to set global variable for flag %s. Error: %v", utils.DataDirFlag.Name, err)
	}

	c := MakeShardingClient(ctx)
	if err := c.Start(); err != nil {
		t.Errorf("Failed to start server: %v", err)
	}
}
