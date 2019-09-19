package p2p

// Config for the p2p service. These parameters are set from application level flags
// to initialize the p2p service.
type Config struct {
	NoDiscovery       bool
	StaticPeers       []string
	BootstrapNodeAddr string
	RelayNodeAddr     string
	HostAddress       string
	PrivateKey        string
	DataDir           string
	TCPPort           uint
	UDPPort           uint
	MaxPeers          uint
	WhitelistCIDR     string
	EnableUPnP        bool
	Encoding          string
}
