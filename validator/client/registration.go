package client

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SubmitValidatorRegistration signs validator registration object and submits it to the beacon node.
func SubmitValidatorRegistration(
	ctx context.Context,
	validatorClient ethpb.BeaconNodeValidatorClient,
	nodeClient ethpb.NodeClient,
	signer signingFunc,
	regs []*ethpb.ValidatorRegistrationV1,
) error {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitBuilderValidatorRegistration")
	defer span.End()

	if len(regs) == 0 {
		return nil
	}
	genesisResponse, err := nodeClient.GetGenesis(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "gRPC call to get genesis time failed")
	}
	ts := time.Unix(int64(regs[0].Timestamp), 0)
	secs := int64(ts.Second()) - genesisResponse.GenesisTime.Seconds
	currentSlot := types.Slot(uint64(secs) / params.BeaconConfig().SecondsPerSlot)

	signedRegs := make([]*ethpb.SignedValidatorRegistrationV1, len(regs))
	for i, reg := range regs {
		sig, err := signValidatorRegistration(ctx, currentSlot, validatorClient, signer, reg)
		if err != nil {
			log.WithError(err).Error("failed to sign builder validator registration obj")
			continue
		}
		signedRegs[i] = &ethpb.SignedValidatorRegistrationV1{
			Message:   reg,
			Signature: sig,
		}
	}

	if _, err := validatorClient.SubmitValidatorRegistration(ctx, &ethpb.SignedValidatorRegistrationsV1{
		Messages: signedRegs,
	}); err != nil {
		return errors.Wrap(err, "could not submit signed registrations to beacon node")
	}

	return nil
}

// Sings validator registration obj with the proposer domain and private key.
func signValidatorRegistration(
	ctx context.Context,
	slot types.Slot,
	validatorClient ethpb.BeaconNodeValidatorClient,
	signer signingFunc,
	reg *ethpb.ValidatorRegistrationV1,
) ([]byte, error) {

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
