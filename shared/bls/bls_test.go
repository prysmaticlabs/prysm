package bls

import (
	"testing"

	"github.com/prysmaticlabs/pbc"

	"crypto/sha256"
)

type data struct {
	msg string
	sig []byte
}

func TestSingleVerification(t *testing.T) {
	// Setting up new pairing
	params := pbc.GenerateA(160, 512)
	pairing := params.NewPairing()
	generator := pairing.NewG2().Rand()

	privateKey1 := pairing.NewZr().Rand()
	pubKey1 := pairing.NewG2().PowZn(generator, privateKey1)

	t.Log(privateKey1.Bytes()) // 20 bytes
	t.Log(pubKey1.Bytes())     // 128 bytes

	// Sign a message, hash it to h
	message := "today is a good day"
	h := pairing.NewG1().SetFromStringHash(message, sha256.New())

	sig1 := pairing.NewG2().PowZn(h, privateKey1)

	// To verify, we check that e(h,g^x)=e(sig,g)
	if !pairing.NewGT().Pair(h, pubKey1).Equals(pairing.NewGT().Pair(sig1, generator)) {
		t.Error("Signature verificaiton failed")
	}
}

func TestAggregateVerification(t *testing.T) {
	// Setting up new pairing
	params := pbc.GenerateA(160, 512)
	pairing := params.NewPairing()
	generator := pairing.NewG2().Rand()

	privateKey1 := pairing.NewZr().Rand()
	privateKey2 := pairing.NewZr().Rand()
	privateKey3 := pairing.NewZr().Rand()

	pubKey1 := pairing.NewG2().PowZn(generator, privateKey1)
	pubKey2 := pairing.NewG2().PowZn(generator, privateKey2)
	pubKey3 := pairing.NewG2().PowZn(generator, privateKey3)

	message := "tomorrow is gonna be awesome"
	h := pairing.NewG1().SetFromStringHash(message, sha256.New())

	sig1 := pairing.NewG2().PowZn(h, privateKey1)
	sig2 := pairing.NewG2().PowZn(h, privateKey2)
	sig3 := pairing.NewG2().PowZn(h, privateKey3)

	// Aggregate all the signatures
	aggregatedSig := pairing.NewG2().Add(pairing.NewG2().Add(sig1, sig2), sig3)

	// Verify them
	tmp1 := pairing.NewGT().Pair(h, pubKey1)
	tmp2 := pairing.NewGT().Pair(h, pubKey2)
	tmp3 := pairing.NewGT().Pair(h, pubKey3)

	check1 := pairing.NewGT().Add(pairing.NewGT().Add(tmp1, tmp2), tmp3)
	check2 := pairing.NewGT().Pair(aggregatedSig, generator)

	// To verify, we check that e(h,g^x)=e(sig,g)
	if !check1.Equals(check2) {
		t.Error("Signature verificaiton failed")
	}
}
