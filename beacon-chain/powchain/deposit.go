package powchain

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
)

func (s *Service) processDeposit(eth1Data *ethpb.Eth1Data, deposit *ethpb.Deposit) error {
	vals := s.preGenesisState.Validators()
	valIndexMap := stateutils.ValidatorIndexMap(vals)
	if err := s.preGenesisState.SetEth1Data(eth1Data); err != nil {
		return err
	}
	return blocks.ProcessPreGenesisDeposit(context.Background(), s.preGenesisState, vals, deposit, valIndexMap)
}
