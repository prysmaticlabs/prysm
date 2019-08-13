package p2p

type Config struct {
	NoDiscovery       bool
	StaticPeers       []string
	BootstrapNodeAddr string
	RelayNodeAddr     string
	HostAddress       string
	PrivateKey        string
	Port              uint
	MaxPeers          uint
	WhitelistCIDR     string
	EnableUPnP        bool
}
