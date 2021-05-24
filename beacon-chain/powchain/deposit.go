package powchain

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
)

func (s *Service) processDeposit(ctx context.Context, eth1Data *ethpb.Eth1Data, deposit *ethpb.Deposit) error {
	var err error
	if err := s.preGenesisState.SetEth1Data(eth1Data); err != nil {
		return err
	}
	beaconState, err := blocks.ProcessPreGenesisDeposits(ctx, s.preGenesisState, []*ethpb.Deposit{deposit})
	if err != nil {
		return errors.Wrap(err, "could not process pre-genesis deposits")
	}
	if beaconState != nil && !beaconState.IsNil() {
		s.preGenesisState = beaconState
	}
	return nil
}
