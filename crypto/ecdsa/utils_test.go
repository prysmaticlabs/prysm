package ecdsa

import (
	"crypto/ecdsa"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestConvertToInterfacePubkey(t *testing.T) {
	privKey, err := gcrypto.GenerateKey()
	require.NoError(t, err)

	pubkey, ok := privKey.Public().(*ecdsa.PublicKey)
	require.NotEqual(t, false, ok)

	altPubkey, err := ConvertToInterfacePubkey(pubkey)
	require.NoError(t, err)

	nKey := *(altPubkey.(*crypto.Secp256k1PublicKey))
	rawKey := btcec.PublicKey(nKey).SerializeUncompressed()
	origRawKey := gcrypto.FromECDSAPub(pubkey)
	assert.DeepEqual(t, origRawKey, rawKey)
}
