package powchain

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func (s *Service) processDeposit(eth1Data *ethpb.Eth1Data, deposit *ethpb.Deposit) error {
	var err error
	valIndexMap := stateutils.ValidatorIndexMap(s.preGenesisState)
	s.preGenesisState.Eth1Data = eth1Data
	s.preGenesisState, err = blocks.ProcessDeposit(s.preGenesisState, deposit, valIndexMap)
	if err != nil {
		return errors.Wrap(err, "could not process deposit")
	}
	pubkey := deposit.Data.PublicKey
	index, ok := valIndexMap[bytesutil.ToBytes48(pubkey)]
	if !ok {
		return nil
	}
	balance := s.preGenesisState.Balances[index]
	s.preGenesisState.Validators[index].EffectiveBalance = mathutil.Min(balance-balance%params.BeaconConfig().EffectiveBalanceIncrement, params.BeaconConfig().MaxEffectiveBalance)
	if s.preGenesisState.Validators[index].EffectiveBalance ==
		params.BeaconConfig().MaxEffectiveBalance {
		s.preGenesisState.Validators[index].ActivationEligibilityEpoch = 0
		s.preGenesisState.Validators[index].ActivationEpoch = 0
	}
	return nil
}
