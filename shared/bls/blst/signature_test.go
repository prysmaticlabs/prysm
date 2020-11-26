// +build linux,amd64 linux,arm64 darwin,amd64 windows,amd64
// +build blst_enabled

package blst

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls/common"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	blst "github.com/supranational/blst/bindings/go"
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
	aggSig := Aggregate(sigs)
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

func TestSignatureFailure(t *testing.T) {
	rawSigs := []string{"0xb6d49a322c63a40fcf97b1388f7df1e313335dcfd43f29f0740b5b033f2c0a74a08e73c4cab02a58604fe902f052f7cf09fbd87b83059c7e64d9ca9ad709e5b9fc475690c72a47132788bc0bd6e2986f1695edc8eb6fdcf66e290ce91c7c53ed", "0xae63b3a828d5cf6bb2125e0eb21f137df08bd939788086aa077a58e041e2fd4891cedade09e9f2436f07c628b8d28a5017529dcc03f67a7d5b342d0582d1a1699d9de72ed74a8bd8948466fe3ce94d6313f9ff7e69f491ad512bddc21c36b8bd", "0x8c04f30d35b057cbfbb98237c238c01ed1fbc29b1160197e05e805d8a35575cbf8307bd4c7d22103b0accf48f2681a860211b352d56d82bc007119a1c58601d3b296a33f090436906e42ef3777e46991d3df04086da738ed057d7b7875f369f1", "0xb0bad3dc528347f911d6e039316c9e5fb8915fe6875a301a58ad614d2b17e618a4673ba1000922d99cfdede94283e43c04e19e5ab5e0efb23b5f70ffaa1097e7250d214c8af452a243656de5495d26092a92f0320fb0adde8fcedf5ef6e5b259", "0xb59f17c00507f36e717170d9d57e0c76bb59440808fe4bdc60d3bd105878bcc4beb56a54822e1110561c537e236bcec4135ae6392013b7ca535219fcd77f8b8b3ed8a5f9978470b55e48672e3fbf3681f56a2486aa88ba05c6afa914aaba3d04", "0xab3275605885ea5b74972015b9df03595c9e329405b22dc1965aa6640795411f877fbb8aa1114c44ca288c9f84e50b7a124e98b355cc50454b457abee778f6fa7830988562cc7226f6c74f931de79023ad1821f1bc2edd53dca0ea9990e8683c"}
	rawPubkeys := []string{"0xb2c9669a6f3e64a5f8b37c210c88adb507ce396042921b1aefd5bf52a3089f0ab0b5695cddf9b1a1157fe5dd43558e54", "0xb2c9669a6f3e64a5f8b37c210c88adb507ce396042921b1aefd5bf52a3089f0ab0b5695cddf9b1a1157fe5dd43558e54", "0xa923d6728c7f14b9bbf628b22b041e32e0e158f0476368bc623c8b812ec57e9459f298aa4071c15440afc861323a81fa", "0x8f0a967a3055d5ae2bf185e25eab44b5d2df65afa5015d238f6030b3261a3e64680cfdc58d6091d5ca699a9f8a0c821b", "0xa25189ebfc72450d40d4c2cdced3b2ec92d04de5dc0a7bd2915a26471db29649b5f14d235aced43d2cceb89644474795", "0x96d3dd0fc08c48cfe6ffde68ed08bde0d69eb30c00dcdfca05df42c6aeafeb92b019d8db875292f50a1b8d8c34f5a5ff"}
	rawMsgs := []string{"0xed9f5dd163af90519a2d2465d94d67ce3a6732ea15fbb2378395c94b4ce01ed9", "0x27faaaa6c7ef2e275cbbc2273d7e03300c11d6b40e3ef221b3f29a94415a0549", "0xf26a0c897bc0f253bd694a3e4bf03cad2571d3b53c45fe65750e542028dc78d4", "0x78d0b38bde21554d73dce32593fc6a88cd768ad6bf414402de7820db5e6394ba", "0xe13824509909401d08640f7ce52de4450e88464b5d78fb940aac0fcd89de32e2", "0xc120a7a0b8e150b7c706b054654e0d236ae2a95e9cefc7a81efcebd0bfb73ca3"}
	sigs := []*blst.P2Affine{}
	pubkeys := []*blst.P1Affine{}
	msgs := [][]byte{}

	for _, s := range rawSigs {
		raw, err := hex.DecodeString(strings.Trim(s, "0x"))
		if err != nil {
			t.Fatal(err)
		}
		signature := new(blstSignature).Uncompress(raw)
		if signature == nil {
			t.Fatal("nil signature returned")
		}
		// Group check signature
		if !signature.SigValidate(false) {
			t.Fatal("signature is invalid")
		}
		sigs = append(sigs, signature)
	}
	for _, p := range rawPubkeys {
		raw, err := hex.DecodeString(strings.Trim(p, "0x"))
		if err != nil {
			t.Fatal(err)
		}
		pubkey := new(blstPublicKey).Uncompress(raw)
		if !pubkey.KeyValidate() {
			t.Fatal("invalid pubkey")
		}
		pubkeys = append(pubkeys, pubkey)
	}
	for _, p := range rawMsgs {
		raw, err := hex.DecodeString(strings.Trim(p, "0x"))
		if err != nil {
			t.Fatal(err)
		}
		msgs = append(msgs, raw)
	}

	// Verify each pairing one by one
	for i, s := range sigs {
		verified := s.Verify(false, pubkeys[i], false, msgs[i], dst)
		if !verified {
			t.Error("signature is not verified")
		}
	}

	// Perform Multiple Signature Verification
	randFunc := func(scalar *blst.Scalar) {
		var rbytes [scalarBytes]byte
		_, err := rand.Reader.Read(rbytes[:])
		_ = err
		scalar.FromBEndian(rbytes[:])
	}

	dummySig := new(blstSignature)

	verified := dummySig.MultipleAggregateVerify(sigs, true, pubkeys, false, msgs, dst, randFunc, randBitsEntropy)
	if !verified {
		t.Error("multi signature verification fails")
	}
}
