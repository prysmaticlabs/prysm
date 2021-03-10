package pandora

import (
	"github.com/ethereum/go-ethereum/rpc"
	"reflect"
	"testing"
)

type mockPandoraService struct{}

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

func (s *mockPandoraService) NoArgsRets() {}

func (s *mockPandoraService) Echo(str string, i int, args *echoArgs) echoResult {
	return echoResult{str, i, args}
}

func NewMockPandoraServer() *rpc.Server {
	server := rpc.NewServer()
	if err := server.RegisterName("pandora", new(mockPandoraService)); err != nil {
		panic(err)
	}
	return server
}

// TODO- Need to add more test cases after finalization of request-response of pandora's apis.
func TestClientRequest(t *testing.T) {
	server := NewMockPandoraServer()
	defer server.Stop()
	client := rpc.DialInProc(server)
	defer client.Close()

	var resp echoResult
	if err := client.Call(&resp, "pandora_echo", "hello", 10, &echoArgs{"world"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resp, echoResult{"hello", 10, &echoArgs{"world"}}) {
		t.Errorf("incorrect result %#v", resp)
	}
}
