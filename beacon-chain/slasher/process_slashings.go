package slasher

import slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"

func (s *Service) processAttesterSlashings(slashings []*slashertypes.Slashing) {
	for _, attSlashing := range slashings {
		preState, err := s.serviceCfg.StateFetcher.AttestationPreState(ctx)
	}
}
