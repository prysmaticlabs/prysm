package debug

import (
	"testing"
)

func TestServer_GetForkChoice(t *testing.T) {
	bs := &Server{
		HeadFetcher: &mockChain.ChainService{State: bs, Root: genesisRoot[:]},
	}

}
