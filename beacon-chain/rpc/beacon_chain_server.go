package rpc

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// BeaconChainServer defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
type BeaconChainServer struct {
	beaconDB *db.BeaconDB
}

// ListValidatorBalances retrieves the validator balances for a given set of public key at
// a specific epoch in time.
//
// TODO(3045): Implement balances at a specific epoch. Current implementation returns latest balances,
// blocked by DB refactor
func (bs *BeaconChainServer) ListValidatorBalances(
	ctx context.Context,
	req *ethpb.GetValidatorBalancesRequest) (*ethpb.ValidatorBalances, error) {

	res := make([]*ethpb.ValidatorBalances_Balance, 0, len(req.PublicKeys)+len(req.Indices))
	filtered := map[uint64]bool{} // track filtered validator to prevent duplication

	balances, err := bs.beaconDB.Balances(ctx)
	if err != nil {
		// TODO: return grpc error
		return nil, err
	}
	validators, err := bs.beaconDB.Validators(ctx)
	if err != nil {
		// TODO: return grpc error
		return nil, err
	}

	for _, pubkey := range req.PublicKeys {
		index, err := bs.beaconDB.ValidatorIndex(pubkey)
		if err != nil {
			// TODO: return grpc error
			return nil, err
		}
		filtered[index] = true

		res = append(res, &ethpb.ValidatorBalances_Balance{
			PublicKey: pubkey,
			Index:     index,
			Balance:   balances[index],
		})
	}

	for _, index := range req.Indices {
		if !filtered[index] {
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: validators[index].PublicKey,
				Index:     index,
				Balance:   balances[index],
			})
		}
	}

	return &ethpb.ValidatorBalances{Balances: res}, nil
}
