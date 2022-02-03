package v1

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func Test_handleRPCError(t *testing.T) {

}

func TestGetPayload(t *testing.T) {
	server := newTestServer()
	defer server.Stop()
	rpcClient := rpc.DialInProc(server)
	defer rpcClient.Close()
	client := &Client{}
	client.rpc = rpcClient

	ctx := context.Background()
	fix := fixtures()
	want := fix["ExecutionPayload"].(*pb.ExecutionPayload)
	payloadId := [8]byte{1}
	resp, err := client.GetPayload(ctx, payloadId)
	require.NoError(t, err)
	require.DeepEqual(t, want, resp)
}

func newTestServer() *rpc.Server {
	server := rpc.NewServer()
	if err := server.RegisterName("engine", new(testEngineService)); err != nil {
		panic(err)
	}
	return server
}

func fixtures() map[string]interface{} {
	foo := bytesutil.ToBytes32([]byte("foo"))
	bar := bytesutil.PadTo([]byte("bar"), 20)
	baz := bytesutil.PadTo([]byte("baz"), 256)
	executionPayloadFixture := &pb.ExecutionPayload{
		ParentHash:    foo[:],
		FeeRecipient:  bar,
		StateRoot:     foo[:],
		ReceiptsRoot:  foo[:],
		LogsBloom:     baz,
		Random:        foo[:],
		BlockNumber:   1,
		GasLimit:      1,
		GasUsed:       1,
		Timestamp:     1,
		ExtraData:     foo[:],
		BaseFeePerGas: foo[:],
		BlockHash:     foo[:],
		Transactions:  [][]byte{foo[:]},
	}
	return map[string]interface{}{
		"ExecutionPayload": executionPayloadFixture,
	}
}

type testEngineService struct{}

type echoArgs struct {
	S string
}

type echoResult struct {
	String string
	Int    int
	Args   *echoArgs
}

type testError struct{}

func (testError) Error() string          { return "testError" }
func (testError) ErrorCode() int         { return 444 }
func (testError) ErrorData() interface{} { return "testError data" }

func (s *testEngineService) NoArgsRets() {}

func (s *testEngineService) GetPayloadV1(
	ctx context.Context, payloadId [8]byte,
) *pb.ExecutionPayload {
	fix := fixtures()
	return fix["ExecutionPayload"].(*pb.ExecutionPayload)
}
