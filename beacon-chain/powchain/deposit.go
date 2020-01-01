package powchain

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
)

func (s *Service) processDeposit(eth1Data *ethpb.Eth1Data, deposit *ethpb.Deposit) error {
	var err error
	valIndexMap := stateutils.ValidatorIndexMap(s.preGenesisState)
	s.preGenesisState.Eth1Data = eth1Data
	s.preGenesisState, err = blocks.ProcessPreGenesisDeposit(context.Background(), s.preGenesisState, deposit, valIndexMap)
	return err
}
