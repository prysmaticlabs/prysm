package powchain

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
)

func (s *Service) processDeposit(eth1Data *ethpb.Eth1Data, deposit *ethpb.Deposit) error {
	var err error
	if err := s.preGenesisState.SetEth1Data(eth1Data); err != nil {
		return err
	}
	s.preGenesisState, err = blocks.ProcessPreGenesisDeposit(context.Background(), s.preGenesisState, []*ethpb.Deposit{deposit}	)
	return err
}
