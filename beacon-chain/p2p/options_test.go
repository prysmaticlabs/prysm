package p2p

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"net"
	"os"
	"testing"

	gethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestPrivateKeyLoading(t *testing.T) {
	file, err := ioutil.TempFile(testutil.TempDir(), "key")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil {
			t.Log(err)
		}
	}()
	key, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		t.Fatalf("Could not generate key: %v", err)
	}
	raw, err := key.Raw()
	if err != nil {
		panic(err)
	}
	out := hex.EncodeToString(raw)

	err = ioutil.WriteFile(file.Name(), []byte(out), params.BeaconIoConfig().ReadWritePermissions)
	if err != nil {
		t.Fatalf("Could not write key to file: %v", err)
	}
	log.WithField("file", file.Name()).WithField("key", out).Info("Wrote key to file")
	cfg := &Config{
		PrivateKey: file.Name(),
	}
	pKey, err := privKey(cfg)
	if err != nil {
		t.Fatalf("Could not apply option: %v", err)
	}
	newPkey := convertToInterfacePrivkey(pKey)
	rawBytes, err := key.Raw()
	if err != nil {
		t.Fatal(err)
	}
	newRaw, err := newPkey.Raw()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newRaw, rawBytes) {
		t.Errorf("Private keys do not match got %#x but wanted %#x", rawBytes, newRaw)
	}
}

func TestIPV6Support(t *testing.T) {
	key, err := gethCrypto.GenerateKey()
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
