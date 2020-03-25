package p2p

import (
	stateFeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
)

// Config for the p2p service. These parameters are set from application level flags
// to initialize the p2p service.
type Config struct {
	StateNotifier         stateFeed.Notifier
	NoDiscovery           bool
	StaticPeers           []string
	BootstrapNodeAddr     []string
	KademliaBootStrapAddr []string
	Discv5BootStrapAddr   []string
	RelayNodeAddr         string
	LocalIP               string
	HostAddress           string
	HostDNS               string
	PrivateKey            string
	DataDir               string
	TCPPort               uint
	UDPPort               uint
	MaxPeers              uint
	WhitelistCIDR         string
	EnableUPnP            bool
	EnableDiscv5          bool
	Encoding              string
}
