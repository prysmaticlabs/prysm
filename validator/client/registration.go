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
func SubmitValidatorRegistration(ctx context.Context, validatorClient ethpb.BeaconNodeValidatorClient, signer signingFunc, reg *ethpb.ValidatorRegistrationV1) error {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitBuilderValidatorRegistration")
	defer span.End()

	sig, err := signValidatorRegistration(ctx, signer, reg)
	if err != nil {
		return errors.Wrap(err, "failed to sign builder validator registration obj")
	}

	signedReg := &ethpb.SignedValidatorRegistrationV1{
		Message:   reg,
		Signature: sig,
	}
	if _, err := validatorClient.SubmitValidatorRegistration(ctx, signedReg); err != nil {
		return errors.Wrap(err, "could not submit signed registration to beacon node")
	}

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
