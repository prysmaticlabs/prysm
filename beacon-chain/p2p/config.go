package p2p

import (
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
)

// Config for the p2p service. These parameters are set from application level flags
// to initialize the p2p service.
type Config struct {
	NoDiscovery         bool
	EnableUPnP          bool
	DisableDiscv5       bool
	StaticPeers         []string
	BootstrapNodeAddr   []string
	Discv5BootStrapAddr []string
	RelayNodeAddr       string
	LocalIP             string
	HostAddress         string
	HostDNS             string
	PrivateKey          string
	DataDir             string
	MetaDataDir         string
	TCPPort             uint
	UDPPort             uint
	MaxPeers            uint
	AllowListCIDR       string
	DenyListCIDR        []string
	StateNotifier       statefeed.Notifier
}
