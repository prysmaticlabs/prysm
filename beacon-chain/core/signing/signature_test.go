package signing_test

import (
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func TestVerifyRegistrationSignature(t *testing.T) {
	sk, err := bls.RandKey()
	require.NoError(t, err)
	reg := &ethpb.ValidatorRegistrationV1{
		FeeRecipient: bytesutil.PadTo([]byte("fee"), 20),
		GasLimit:     123456,
		Timestamp:    uint64(time.Now().Unix()),
		Pubkey:       sk.PublicKey().Marshal(),
	}
	st, _ := util.DeterministicGenesisState(t, 1)
	d := params.BeaconConfig().DomainApplicationBuilder
	e := slots.ToEpoch(st.Slot())
	sig, err := signing.ComputeDomainAndSign(st, e, reg, d, sk)
	require.NoError(t, err)
	sReg := &ethpb.SignedValidatorRegistrationV1{
		Message:   reg,
		Signature: sig,
	}
	f := st.Fork()
	g := st.GenesisValidatorsRoot()
	require.NoError(t, signing.VerifyRegistrationSignature(e, f, sReg, g))

	sReg.Signature = []byte("bad")
	require.ErrorIs(t, signing.VerifyRegistrationSignature(e, f, sReg, g), signing.ErrSigFailedToVerify)

	sReg.Message = nil
	require.ErrorIs(t, signing.VerifyRegistrationSignature(e, f, sReg, g), signing.ErrNilRegistration)
}
