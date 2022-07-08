package client

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"go.opencensus.io/trace"
)

// SubmitValidatorRegistration signs validator registration object and submits it to the beacon node.
func SubmitValidatorRegistration(
	ctx context.Context,
	validatorClient ethpb.BeaconNodeValidatorClient,
	signedRegs []*ethpb.SignedValidatorRegistrationV1,
) error {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitBuilderValidatorRegistration")
	defer span.End()

	if len(signedRegs) == 0 {
		return nil
	}

	if _, err := validatorClient.SubmitValidatorRegistration(ctx, &ethpb.SignedValidatorRegistrationsV1{
		Messages: signedRegs,
	}); err != nil {
		return errors.Wrap(err, "could not submit signed registrations to beacon node")
	}
	log.Infoln("Submitted builder validator registration settings for custom builders")
	return nil
}

// Sings validator registration obj with the proposer domain and private key.
func signValidatorRegistration(ctx context.Context, signer signingFunc, reg *ethpb.ValidatorRegistrationV1) ([]byte, error) {

	// Per spec, we want the fork version and genesis validator to be nil.
	// Which is genesis value and zero by default.
	d, err := signing.ComputeDomain(
		params.BeaconConfig().DomainApplicationBuilder,
		nil, /* fork version */
		nil /* genesis val root */)
	if err != nil {
		return nil, err
	}

	r, err := signing.ComputeSigningRoot(reg, d)
	if err != nil {
		return nil, errors.Wrap(err, signingRootErr)
	}

	sig, err := signer(ctx, &validatorpb.SignRequest{
		PublicKey:       reg.Pubkey,
		SigningRoot:     r[:],
		SignatureDomain: d,
		Object:          &validatorpb.SignRequest_Registration{Registration: reg},
	})
	if err != nil {
		return nil, errors.Wrap(err, signExitErr)
	}
	return sig.Marshal(), nil
}
