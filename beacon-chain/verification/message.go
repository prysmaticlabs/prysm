package verification

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	p2ptypes "github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/types"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

type MsgVerifError struct {
	PubsubResult pubsub.ValidationResult
	Err          error
}

func SignedContributionAndProofValidationSetup(ctx context.Context, headFetcher blockchain.HeadFetcher, req *ethpb.SignedContributionAndProof) ([fieldparams.RootLength]byte, bls.PublicKey, *MsgVerifError) {
	ctx, span := trace.StartSpan(ctx, "verification.SignedContributionAndProofValidationSetup")
	defer span.End()
	// The aggregate signature is valid for the message `beacon_block_root` and aggregate pubkey
	// derived from the participation info in `aggregation_bits` for the subcommittee specified by the `contribution.subcommittee_index`.
	var activeRawPubkeys [][]byte
	syncPubkeys, err := headFetcher.HeadSyncCommitteePubKeys(ctx, req.Message.Contribution.Slot, primitives.CommitteeIndex(req.Message.Contribution.SubcommitteeIndex))
	if err != nil {
		tracing.AnnotateError(span, err)
		return [fieldparams.RootLength]byte{}, nil, &MsgVerifError{PubsubResult: pubsub.ValidationIgnore, Err: err}
	}
	bVector := req.Message.Contribution.AggregationBits
	// In the event no bit is set for the
	// sync contribution, we reject the message.
	if bVector.Count() == 0 {
		tracing.AnnotateError(span, err)
		return [fieldparams.RootLength]byte{}, nil, &MsgVerifError{PubsubResult: pubsub.ValidationReject, Err: errors.New("bitvector count is 0")}
	}
	for i, pk := range syncPubkeys {
		if bVector.BitAt(uint64(i)) {
			activeRawPubkeys = append(activeRawPubkeys, pk)
		}
	}
	d, err := headFetcher.HeadSyncCommitteeDomain(ctx, req.Message.Contribution.Slot)
	if err != nil {
		tracing.AnnotateError(span, err)
		return [fieldparams.RootLength]byte{}, nil, &MsgVerifError{PubsubResult: pubsub.ValidationIgnore, Err: err}
	}
	rawBytes := p2ptypes.SSZBytes(req.Message.Contribution.BlockRoot)
	sigRoot, err := signing.ComputeSigningRoot(&rawBytes, d)
	if err != nil {
		tracing.AnnotateError(span, err)
		return [fieldparams.RootLength]byte{}, nil, &MsgVerifError{PubsubResult: pubsub.ValidationIgnore, Err: err}
	}
	// Aggregate pubkeys separately again to allow
	// for signature sets to be created for batch verification.
	aggKey, err := bls.AggregatePublicKeys(activeRawPubkeys)
	if err != nil {
		tracing.AnnotateError(span, err)
		return [fieldparams.RootLength]byte{}, nil, &MsgVerifError{PubsubResult: pubsub.ValidationIgnore, Err: err}
	}
	return sigRoot, aggKey, nil
}
