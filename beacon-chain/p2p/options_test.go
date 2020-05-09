package p2p

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
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

	err = ioutil.WriteFile(file.Name(), []byte(out), 0600)
	if err != nil {
		t.Fatalf("Could not write key to file: %v", err)
	}
	log.WithField("file", file.Name()).WithField("key", out).Info("Wrote key to file")
	cfg := &Config{
		PrivateKey: file.Name(),
		Encoding:   "ssz",
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

func TestPeerBlacklist(t *testing.T) {
	// create host with blacklist
	ipAddr, pkey := createAddrAndPrivKey(t)
	ipAddr2, pkey2 := createAddrAndPrivKey(t)

	mask := ipAddr2.DefaultMask()
	ones, _ := mask.Size()
	maskedIP := ipAddr2.Mask(mask)
	cidr := maskedIP.String() + fmt.Sprintf("/%d", ones)

	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, 2000))
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	h1, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey), libp2p.ListenAddrs(listen), blacklistSubnets([]string{cidr})}...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := h1.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// create alternate host
	listen, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr2, 3000))
	if err != nil {
		t.Fatalf("Failed to p2p listen: %v", err)
	}
	h2, err := libp2p.New(context.Background(), []libp2p.Option{privKeyOption(pkey2), libp2p.ListenAddrs(listen)}...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := h2.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	multiAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipAddr2, 3000, h2.ID()))
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddress)
	if err != nil {
		t.Fatal(err)
	}
	err = h1.Connect(context.Background(), *addrInfo)
	if err == nil {
		t.Error("Wanted connection to fail with blacklist")
	}
}
