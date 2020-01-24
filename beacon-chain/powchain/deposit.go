package powchain

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
)

func (s *Service) processDeposit(eth1Data *ethpb.Eth1Data, deposit *ethpb.Deposit) error {
	var err error
	s.preGenesisState.SetEth1Data(eth1Data)
	s.preGenesisState, err = blocks.ProcessPreGenesisDeposit(context.Background(), s.preGenesisState, deposit)
	return err
}
