package herumi_test

import (
	"testing"

	"github.com/herumi/bls-eth-go-binary/bls"
	"github.com/prysmaticlabs/prysm/shared/bls/common"
	"github.com/prysmaticlabs/prysm/shared/bls/herumi"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func BenchmarkPairing(b *testing.B) {
	require.NoError(b, bls.Init(bls.BLS12_381))
	if err := bls.SetETHmode(bls.EthModeDraft07); err != nil {
		panic(err)
	}
	newGt := &bls.GT{}
	newG1 := &bls.G1{}
	newG2 := &bls.G2{}

	newGt.SetInt64(10)
	hash := hashutil.Hash([]byte{})
	require.NoError(b, newG1.HashAndMapTo(hash[:]))
	require.NoError(b, newG2.HashAndMapTo(hash[:]))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bls.Pairing(newGt, newG1, newG2)
	}

}
func BenchmarkSignature_Verify(b *testing.B) {
	sk, err := herumi.RandKey()
	require.NoError(b, err)

	msg := []byte("Some msg")
	sig := sk.Sign(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.Equal(b, true, sig.Verify(sk.PublicKey(), msg))
	}
}

func BenchmarkSignature_AggregateVerify(b *testing.B) {
	sigN := 128 // MAX_ATTESTATIONS per block.

	var pks []common.PublicKey
	var sigs []common.Signature
	var msgs [][32]byte
	for i := 0; i < sigN; i++ {
		msg := [32]byte{'s', 'i', 'g', 'n', 'e', 'd', byte(i)}
		sk, err := herumi.RandKey()
		require.NoError(b, err)
		sig := sk.Sign(msg[:])
		pks = append(pks, sk.PublicKey())
		sigs = append(sigs, sig)
		msgs = append(msgs, msg)
	}
	aggregated := herumi.Aggregate(sigs)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		require.Equal(b, true, aggregated.AggregateVerify(pks, msgs))
	}
}

func BenchmarkSecretKey_Marshal(b *testing.B) {
	key, err := herumi.RandKey()
	require.NoError(b, err)
	d := key.Marshal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := herumi.SecretKeyFromBytes(d)
		_ = err
	}
}
