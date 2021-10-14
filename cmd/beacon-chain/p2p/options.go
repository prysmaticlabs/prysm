package p2pcmd

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/node/registration"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/container/slice"
	"github.com/urfave/cli/v2"
)

// Options for peer-to-peer networking configurations.
func Options(c *cli.Context) ([]p2p.Option, error) {
	bootstrapNodeAddrs, dataDir, err := registration.P2PPreregistration(c)
	if err != nil {
		return nil, err
	}
	opts := []p2p.Option{
		p2p.WithStaticPeers(
			slice.SplitCommaSeparated(c.StringSlice(StaticPeers.Name)),
		),
		p2p.WithBootstrapNodeAddr(bootstrapNodeAddrs),
		p2p.WithRelayNodeAddr(c.String(RelayNode.Name)),
		p2p.WithDataDir(dataDir),
		p2p.WithLocalIP(c.String(P2PIP.Name)),
		p2p.WithHostAddr(c.String(P2PHost.Name)),
		p2p.WithHostDNS(c.String(P2PHostDNS.Name)),
		p2p.WithPrivateKey(c.String(P2PPrivKey.Name)),
		p2p.WithMetadataDir(c.String(P2PMetadata.Name)),
		p2p.WithTCPPort(c.Uint(P2PTCPPort.Name)),
		p2p.WithUDPPort(c.Uint(P2PUDPPort.Name)),
		p2p.WithMaxPeers(c.Uint(flags.P2PMaxPeers.Name)),
		p2p.WithAllowListCIDR(c.String(P2PAllowList.Name)),
		p2p.WithDenyListCIDR(
			slice.SplitCommaSeparated(c.StringSlice(P2PDenyList.Name)),
		),
	}
	if c.Bool(NoDiscovery.Name) {
		opts = append(opts, p2p.WithNoDiscovery())
	}
	if c.Bool(EnableUPnPFlag.Name) {
		opts = append(opts, p2p.WithEnableUPnP())
	}
	if c.Bool(DisableDiscv5.Name) {
		opts = append(opts, p2p.WithDisableDiscv5())
	}
	return opts, nil
}
