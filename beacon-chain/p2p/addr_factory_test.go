package p2p

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestRelayAddrs_OnlyFactory(t *testing.T) {
	relay := "/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi"
	f := withRelayAddrs(relay)

	a, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/33201/p2p/QmaXZhW44pwQxBSeLkE5FNeLz8tGTTEsRciFg1DNWXXrWG")
	require.NoError(t, err)
	addrs := []ma.Multiaddr{a}

	result := f(addrs)
	assert.Equal(t, 2, len(result), "Unexpected number of addresses")

	expected := "/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi/p2p-circuit/ip4/127.0.0.1/tcp/33201/p2p/QmaXZhW44pwQxBSeLkE5FNeLz8tGTTEsRciFg1DNWXXrWG"
	if result[1].String() != expected {
		t.Errorf("Address at index 1 (%s) is not the expected p2p-circuit address", result[1].String())
	}
}

func TestRelayAddrs_UseNonRelayAddrs(t *testing.T) {
	relay := "/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi"
	f := withRelayAddrs(relay)

	expected := []string{
		"/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi/p2p-circuit/ip4/127.0.0.1/tcp/33201/p2p/QmaXZhW44pwQxBSeLkE5FNeLz8tGTTEsRciFg1DNWXXrWG",
		"/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi/p2p-circuit/ip4/127.0.0.1/tcp/33203/p2p/QmaXZhW44pwQxBSeLkE5FNeLz8tGTTEsRciFg1DNWXXrWG",
	}

	addrs := make([]ma.Multiaddr, len(expected))
	for i, addr := range expected {
		a, err := ma.NewMultiaddr(addr)
		if err != nil {
			t.Fatal(err)
		}
		addrs[i] = a
	}

	result := f(addrs)
	assert.Equal(t, 2, len(result), "Unexpected number of addresses")
	assert.DeepEqual(t, result, addrs)
}
