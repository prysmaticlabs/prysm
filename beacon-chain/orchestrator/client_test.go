package orchestrator_test

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/orchestrator"
	vanTypes "github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"net"
	"testing"
)

type orchestratorTestService struct{}

func (apiMock *orchestratorTestService) ConfirmVanBlockHashes(args []*orchestrator.BlockHash) (reply []*orchestrator.BlockStatus) {
	for _, singleArg := range args {
		exampleResp := &orchestrator.BlockStatus{
			BlockHash: orchestrator.BlockHash{
				Slot: singleArg.Slot,
				Hash: singleArg.Hash,
			},
			Status: "Verified",
		}
		reply = append(reply, exampleResp)
	}

	return reply
}

func newTestServer() *rpc.Server {
	newServer := rpc.NewServer()
	if err := newServer.RegisterName("orc", new(orchestratorTestService)); err != nil {
		panic(err)
	}

	return newServer
}

func TestRPCClient_ConfirmVanBlockHashes(t *testing.T) {
	// Configure mocked rpcServer
	ctx := context.Background()
	rpcServer := newTestServer()

	defer rpcServer.Stop()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("can't listen: %s", err.Error())
	}

	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}(listener)

	go func() {
		err := rpcServer.ServeListener(listener)
		if err != nil {
			fmt.Println(err.Error())
		}
	}()

	// Configure rpcClient
	orcRpcClient, err := orchestrator.DialInProc(rpcServer)
	assert.NoError(t, err)

	defer orcRpcClient.Close()

	// Perform tests
	exampleHash := common.HexToHash("0xfe88c94d860f01a17f961bf4bdfb6e0c6cd10d3fda5cc861e805ca1240c58553")

	// Request
	blockHashes := make([]*vanTypes.ConfirmationReqData, 1)
	blockHash := &vanTypes.ConfirmationReqData{
		Slot: 0,
		Hash: exampleHash,
	}
	blockHashes[0] = blockHash

	// Response
	expectedBlockStatuses := make([]*vanTypes.ConfirmationResData, 1)
	expectedBlockStatus := &vanTypes.ConfirmationResData{
		Slot:   0,
		Hash:   exampleHash,
		Status: "Verified",
	}
	expectedBlockStatuses[0] = expectedBlockStatus

	// Test cases
	t.Run("connection success and returned with 1 verified block", func(t *testing.T) {
		blockStatuses, err := orcRpcClient.ConfirmVanBlockHashes(ctx, blockHashes)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(blockStatuses))
		assert.DeepEqual(t, expectedBlockStatuses, blockStatuses)
	})

	t.Run("connection success and empty request", func(t *testing.T) {
		emptyBlockHashes := make([]*vanTypes.ConfirmationReqData, 1)
		blockStatuses, err := orcRpcClient.ConfirmVanBlockHashes(ctx, emptyBlockHashes)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(blockStatuses))
		assert.DeepEqual(t, []*vanTypes.ConfirmationResData(nil), blockStatuses)
	})
}
