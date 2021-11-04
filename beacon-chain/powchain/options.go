package powchain

type Option func(s *Service) error

//// Web3ServiceConfig defines a config struct for web3 service to use through its life cycle.
//type Web3ServiceConfig struct {
//	HttpEndpoints          []string
//	DepositContract        common.Address
//	BeaconDB               db.HeadAccessDatabase
//	DepositCache           *depositcache.DepositCache
//	StateNotifier          statefeed.Notifier
//	StateGen               *stategen.State
//	Eth1HeaderReqLimit     uint64
//	BeaconNodeStatsUpdater BeaconNodeStatsUpdater
//}

// WithMaxGoroutines to control resource use of the blockchain service.
func WithMaxGoroutines(x int) Option {
	return func(s *Service) error {
		return nil
	}
}
