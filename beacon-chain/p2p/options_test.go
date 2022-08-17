package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"os"
	"testing"

	gethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	ecdsaprysm "github.com/prysmaticlabs/prysm/v3/crypto/ecdsa"
	"github.com/prysmaticlabs/prysm/v3/network"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestPrivateKeyLoading(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	file, err := os.CreateTemp(t.TempDir(), "key")
	require.NoError(t, err)
	key, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err, "Could not generate key")
	raw, err := key.Raw()
	if err != nil {
		panic(err)
	}
	out := hex.EncodeToString(raw)

	err = os.WriteFile(file.Name(), []byte(out), params.BeaconIoConfig().ReadWritePermissions)
	require.NoError(t, err, "Could not write key to file")
	log.WithField("file", file.Name()).WithField("key", out).Info("Wrote key to file")
	cfg := &Config{
		PrivateKey: file.Name(),
	}
	pKey, err := privKey(cfg)
	require.NoError(t, err, "Could not apply option")
	newPkey, err := ecdsaprysm.ConvertToInterfacePrivkey(pKey)
	require.NoError(t, err)
	rawBytes, err := key.Raw()
	require.NoError(t, err)
	newRaw, err := newPkey.Raw()
	require.NoError(t, err)
	assert.DeepEqual(t, rawBytes, newRaw, "Private keys do not match")
}

func TestIPV6Support(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	key, err := gethCrypto.GenerateKey()
	require.NoError(t, err)
	db, err := enode.OpenDB("")
	if err != nil {
		log.Error("could not open node's peer database")
	}
	lNode := enode.NewLocalNode(db, key)
	mockIPV6 := net.IP{0xff, 0x02, 0xAA, 0, 0x1F, 0, 0x2E, 0, 0, 0x36, 0x45, 0, 0, 0, 0, 0x02}
	lNode.Set(enr.IP(mockIPV6))
	ma, err := convertToSingleMultiAddr(lNode.Node())
	if err != nil {
		t.Fatal(err)
	}
	ipv6Exists := false
	for _, p := range ma.Protocols() {
		if p.Name == "ip4" {
			t.Error("Got ip4 address instead of ip6")
		}
		if p.Name == "ip6" {
			ipv6Exists = true
		}
	}
	if !ipv6Exists {
		t.Error("Multiaddress did not have ipv6 protocol")
	}
}

func TestDefaultMultiplexers(t *testing.T) {
	var cfg libp2p.Config
	_ = cfg
	p2pCfg := &Config{
		TCPPort:       2000,
		UDPPort:       2000,
		StateNotifier: &mock.MockStateNotifier{},
	}
	svc := &Service{cfg: p2pCfg}
	var err error
	svc.privKey, err = privKey(svc.cfg)
	assert.NoError(t, err)
	ipAddr := network.IPAddr()
	opts := svc.buildOptions(ipAddr, svc.privKey)
	err = cfg.Apply(append(opts, libp2p.FallbackDefaults)...)
	assert.NoError(t, err)

	assert.Equal(t, "/mplex/6.7.0", cfg.Muxers[0].ID)
	assert.Equal(t, "/yamux/1.0.0", cfg.Muxers[1].ID)

}
