package sharding

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"sync"
	"testing"
)

// FakeEthService based on implementation of internal/ethapi.Client
type FakeEthService struct {
	mu sync.Mutex

	getCodeResp hexutil.Bytes
	getCodeErr  error
}

func TestSubscribeHeaders(t *testing.T) {

}
