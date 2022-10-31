package p2p

import (
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
)

// Config for the p2p service. These parameters are set from application level flags
// to initialize the p2p service.
type Config struct {
	DB                  db.ReadOnlyDatabase
	StateNotifier       statefeed.Notifier
	DataDir             string
	AllowListCIDR       string
	MetaDataDir         string
	RelayNodeAddr       string
	LocalIP             string
	HostAddress         string
	HostDNS             string
	PrivateKey          string
	Discv5BootStrapAddr []string
	BootstrapNodeAddr   []string
	DenyListCIDR        []string
	StaticPeers         []string
	TCPPort             uint
	UDPPort             uint
	MaxPeers            uint
	NoDiscovery         bool
	EnableUPnP          bool
}
