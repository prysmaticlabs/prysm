package builder

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func VerifyRegistrationSignature(
	slot types.Slot,
	fork *ethpb.Fork,
	signed *ethpb.SignedValidatorRegistrationV1,
	genesisRoot []byte,
) error {
	if signed == nil || signed.Message == nil {
		return errors.New("nil signed registration")
	}
	domain, err := signing.Domain(fork, slots.ToEpoch(slot), [4]byte{} /*TODO: Use registration signing domain */, genesisRoot)
	if err != nil {
		return err
	}

	if err := signing.VerifySigningRoot(signed, signed.Message.Pubkey, signed.Signature, domain); err != nil {
		return signing.ErrSigFailedToVerify
	}
	return nil
}
