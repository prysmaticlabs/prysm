package signing

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

var ErrNilRegistration = errors.New("nil signed registration")

// VerifyRegistrationSignature verifies the signature of a validator's registration.
func VerifyRegistrationSignature(
	e types.Epoch,
	f *ethpb.Fork,
	sr *ethpb.SignedValidatorRegistrationV1,
	genesisRoot []byte,
) error {
	if sr == nil || sr.Message == nil {
		return ErrNilRegistration
	}

	d := params.BeaconConfig().DomainApplicationBuilder
	sd, err := Domain(f, e, d, genesisRoot)
	if err != nil {
		return err
	}

	if err := VerifySigningRoot(sr.Message, sr.Message.Pubkey, sr.Signature, sd); err != nil {
		return ErrSigFailedToVerify
	}
	return nil
}
