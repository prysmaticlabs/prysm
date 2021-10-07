//go:build ((linux && amd64) || (linux && arm64) || (darwin && amd64) || (darwin && arm64) || (windows && amd64)) && !blst_disabled
// +build linux,amd64 linux,arm64 darwin,amd64 darwin,arm64 windows,amd64
// +build !blst_disabled

package blst

import (
	"bytes"
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestSignVerify(t *testing.T) {
	priv, err := RandKey()
	require.NoError(t, err)
	pub := priv.PublicKey()
	msg := []byte("hello")
	sig := priv.Sign(msg)
	assert.Equal(t, true, sig.Verify(pub, msg), "Signature did not verify")
}

func TestAggregateVerify(t *testing.T) {
	pubkeys := make([]common.PublicKey, 0, 100)
	sigs := make([]common.Signature, 0, 100)
	var msgs [][32]byte
	for i := 0; i < 100; i++ {
		msg := [32]byte{'h', 'e', 'l', 'l', 'o', byte(i)}
		priv, err := RandKey()
		require.NoError(t, err)
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:])
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}
	aggSig := AggregateSignatures(sigs)
	assert.Equal(t, true, aggSig.AggregateVerify(pubkeys, msgs), "Signature did not verify")
}

func TestFastAggregateVerify(t *testing.T) {
	pubkeys := make([]common.PublicKey, 0, 100)
	sigs := make([]common.Signature, 0, 100)
	msg := [32]byte{'h', 'e', 'l', 'l', 'o'}
	for i := 0; i < 100; i++ {
		priv, err := RandKey()
		require.NoError(t, err)
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:])
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
	}
	aggSig := AggregateSignatures(sigs)
	assert.Equal(t, true, aggSig.FastAggregateVerify(pubkeys, msg), "Signature did not verify")

}

func TestVerifyCompressed(t *testing.T) {
	priv, err := RandKey()
	require.NoError(t, err)
	pub := priv.PublicKey()
	msg := []byte("hello")
	sig := priv.Sign(msg)
	assert.Equal(t, true, sig.Verify(pub, msg), "Non compressed signature did not verify")
	assert.Equal(t, true, VerifyCompressed(sig.Marshal(), pub.Marshal(), msg), "Compressed signatures and pubkeys did not verify")
}

func TestMultipleSignatureVerification(t *testing.T) {
	pubkeys := make([]common.PublicKey, 0, 100)
	sigs := make([][]byte, 0, 100)
	var msgs [][32]byte
	for i := 0; i < 100; i++ {
		msg := [32]byte{'h', 'e', 'l', 'l', 'o', byte(i)}
		priv, err := RandKey()
		require.NoError(t, err)
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:]).Marshal()
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}
	verify, err := VerifyMultipleSignatures(sigs, msgs, pubkeys)
	assert.NoError(t, err, "Signature did not verify")
	assert.Equal(t, true, verify, "Signature did not verify")
}

func TestFastAggregateVerify_ReturnsFalseOnEmptyPubKeyList(t *testing.T) {
	var pubkeys []common.PublicKey
	msg := [32]byte{'h', 'e', 'l', 'l', 'o'}

	aggSig := NewAggregateSignature()
	assert.Equal(t, false, aggSig.FastAggregateVerify(pubkeys, msg), "Expected FastAggregateVerify to return false with empty input ")
}

func TestEth2FastAggregateVerify(t *testing.T) {
	pubkeys := make([]common.PublicKey, 0, 100)
	sigs := make([]common.Signature, 0, 100)
	msg := [32]byte{'h', 'e', 'l', 'l', 'o'}
	for i := 0; i < 100; i++ {
		priv, err := RandKey()
		require.NoError(t, err)
		pub := priv.PublicKey()
		sig := priv.Sign(msg[:])
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
	}
	aggSig := AggregateSignatures(sigs)
	assert.Equal(t, true, aggSig.Eth2FastAggregateVerify(pubkeys, msg), "Signature did not verify")

}

func TestEth2FastAggregateVerify_ReturnsFalseOnEmptyPubKeyList(t *testing.T) {
	var pubkeys []common.PublicKey
	msg := [32]byte{'h', 'e', 'l', 'l', 'o'}

	aggSig := NewAggregateSignature()
	assert.Equal(t, false, aggSig.Eth2FastAggregateVerify(pubkeys, msg), "Expected Eth2FastAggregateVerify to return false with empty input ")
}

func TestEth2FastAggregateVerify_ReturnsTrueOnG2PointAtInfinity(t *testing.T) {
	var pubkeys []common.PublicKey
	msg := [32]byte{'h', 'e', 'l', 'l', 'o'}

	g2PointAtInfinity := append([]byte{0xC0}, make([]byte, 95)...)
	aggSig, err := SignatureFromBytes(g2PointAtInfinity)
	require.NoError(t, err)
	assert.Equal(t, true, aggSig.Eth2FastAggregateVerify(pubkeys, msg))
}

func TestSignatureFromBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		err   error
	}{
		{
			name: "Nil",
			err:  errors.New("signature must be 96 bytes"),
		},
		{
			name:  "Empty",
			input: []byte{},
			err:   errors.New("signature must be 96 bytes"),
		},
		{
			name:  "Short",
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			err:   errors.New("signature must be 96 bytes"),
		},
		{
			name:  "Long",
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			err:   errors.New("signature must be 96 bytes"),
		},
		{
			name:  "Bad",
			input: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			err:   errors.New("could not unmarshal bytes into signature"),
		},
		{
			name:  "Good",
			input: []byte{0xab, 0xb0, 0x12, 0x4c, 0x75, 0x74, 0xf2, 0x81, 0xa2, 0x93, 0xf4, 0x18, 0x5c, 0xad, 0x3c, 0xb2, 0x26, 0x81, 0xd5, 0x20, 0x91, 0x7c, 0xe4, 0x66, 0x65, 0x24, 0x3e, 0xac, 0xb0, 0x51, 0x00, 0x0d, 0x8b, 0xac, 0xf7, 0x5e, 0x14, 0x51, 0x87, 0x0c, 0xa6, 0xb3, 0xb9, 0xe6, 0xc9, 0xd4, 0x1a, 0x7b, 0x02, 0xea, 0xd2, 0x68, 0x5a, 0x84, 0x18, 0x8a, 0x4f, 0xaf, 0xd3, 0x82, 0x5d, 0xaf, 0x6a, 0x98, 0x96, 0x25, 0xd7, 0x19, 0xcc, 0xd2, 0xd8, 0x3a, 0x40, 0x10, 0x1f, 0x4a, 0x45, 0x3f, 0xca, 0x62, 0x87, 0x8c, 0x89, 0x0e, 0xca, 0x62, 0x23, 0x63, 0xf9, 0xdd, 0xb8, 0xf3, 0x67, 0xa9, 0x1e, 0x84},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := SignatureFromBytes(test.input)
			if test.err != nil {
				assert.NotEqual(t, nil, err, "No error returned")
				assert.ErrorContains(t, test.err.Error(), err, "Unexpected error returned")
			} else {
				assert.NoError(t, err)
				assert.DeepEqual(t, 0, bytes.Compare(res.Marshal(), test.input))
			}
		})
	}
}

func TestCopy(t *testing.T) {
	priv, err := RandKey()
	require.NoError(t, err)
	key, ok := priv.(*bls12SecretKey)
	require.Equal(t, true, ok)

	signatureA := &Signature{s: new(blstSignature).Sign(key.p, []byte("foo"), dst)}
	signatureB, ok := signatureA.Copy().(*Signature)
	require.Equal(t, true, ok)

	assert.NotEqual(t, signatureA, signatureB)
	assert.NotEqual(t, signatureA.s, signatureB.s)
	assert.DeepEqual(t, signatureA, signatureB)

	signatureA.s.Sign(key.p, []byte("bar"), dst)
	assert.DeepNotEqual(t, signatureA, signatureB)
}
