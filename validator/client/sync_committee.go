package client

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func (v *validator) signSelectionData(ctx context.Context, pubKey [48]byte, index uint64, slot types.Slot) ([]byte, error) {
	domain, err := v.domainData(ctx, helpers.SlotToEpoch(slot), params.BeaconConfig().DomainSyncCommitteeSelectionProof[:])
	if err != nil {
		return nil, err
	}

	data := &pb.SyncAggregatorSelectionData{
		Slot:              slot,
		SubcommitteeIndex: index,
	}
	root, err := helpers.ComputeSigningRoot(data, domain.SignatureDomain)
	if err != nil {
		return nil, err
	}
	sig, err := v.keyManager.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey[:],
		SigningRoot:     root[:],
		SignatureDomain: domain.SignatureDomain,
	})
	if err != nil {
		return nil, err
	}

	return sig.Marshal(), nil
}
