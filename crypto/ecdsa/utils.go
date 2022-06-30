package ecdsa

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/pkg/errors"
)

func ConvertFromInterfacePrivKey(privkey crypto.PrivKey) (*ecdsa.PrivateKey, error) {
	secpKey := (privkey.(*crypto.Secp256k1PrivateKey))
	rawKey, err := secpKey.Raw()
	if err != nil {
		return nil, err
	}
	privKey := new(ecdsa.PrivateKey)
	k := new(big.Int).SetBytes(rawKey)
	privKey.D = k
	privKey.Curve = gcrypto.S256() // Temporary hack, so libp2p Secp256k1 is recognized as geth Secp256k1 in disc v5.1.
	privKey.X, privKey.Y = gcrypto.S256().ScalarBaseMult(rawKey)
	return privKey, nil
}

func ConvertToInterfacePrivkey(privkey *ecdsa.PrivateKey) (crypto.PrivKey, error) {
	return crypto.UnmarshalSecp256k1PrivateKey(privkey.D.Bytes())
}

func ConvertToInterfacePubkey(pubkey *ecdsa.PublicKey) (crypto.PubKey, error) {
	if !pubkey.Curve.IsOnCurve(pubkey.X, pubkey.Y) {
		return nil, errors.Errorf("not on curve")
	}
	xVal, yVal := new(btcec.FieldVal), new(btcec.FieldVal)
	if xVal.SetByteSlice(pubkey.X.Bytes()) {
		return nil, errors.Errorf("X value overflows")
	}
	if yVal.SetByteSlice(pubkey.Y.Bytes()) {
		return nil, errors.Errorf("Y value overflows")
	}
	typeAssertedKey := crypto.PubKey((*crypto.Secp256k1PublicKey)(btcec.NewPublicKey(xVal, yVal)))
	return typeAssertedKey, nil
}
