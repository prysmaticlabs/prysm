package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
	status "github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/urfave/cli"
)

var (
	// Required fields
	datadir = flag.String("datadir", "~/data", "Path to data directory.")
)

func init() {
	fc := featureconfig.Get()
	featureconfig.Init(fc)
}

func main() {
	flag.Parse()
	fmt.Println("Starting process...")
	cfg := &kv.Config{}
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("p2p-encoding", "ssz", "p2p encoding scheme")
	ct := cli.NewContext(app, set, nil)
	d, err := db.NewDB(*datadir, cfg)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	slashings, err := d.AttesterSlashings(ctx, status.Active)
	if err != nil {
		panic(err)
	}
	p2p, err := startP2P(ct)
	if err != nil {
		panic(err)
	}
	for _, slashing := range slashings {
		err := p2p.Broadcast(ctx, slashing)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("done")
}

func startP2P(ctx *cli.Context) (*p2p.Service, error) {
	// Bootnode ENR may be a filepath to an ENR file.
	bootnodeAddrs := strings.Split(ctx.GlobalString(cmd.BootstrapNode.Name), ",")
	for i, addr := range bootnodeAddrs {
		if filepath.Ext(addr) == ".enr" {
			b, err := ioutil.ReadFile(addr)
			if err != nil {
				return nil, err
			}
			bootnodeAddrs[i] = string(b)
		}
	}

	svc, err := p2p.NewService(&p2p.Config{
		NoDiscovery:       ctx.GlobalBool(cmd.NoDiscovery.Name),
		StaticPeers:       sliceutil.SplitCommaSeparated(ctx.GlobalStringSlice(cmd.StaticPeers.Name)),
		BootstrapNodeAddr: bootnodeAddrs,
		RelayNodeAddr:     ctx.GlobalString(cmd.RelayNode.Name),
		DataDir:           ctx.GlobalString(cmd.DataDirFlag.Name),
		LocalIP:           ctx.GlobalString(cmd.P2PIP.Name),
		HostAddress:       ctx.GlobalString(cmd.P2PHost.Name),
		HostDNS:           ctx.GlobalString(cmd.P2PHostDNS.Name),
		PrivateKey:        ctx.GlobalString(cmd.P2PPrivKey.Name),
		TCPPort:           ctx.GlobalUint(cmd.P2PTCPPort.Name),
		UDPPort:           ctx.GlobalUint(cmd.P2PUDPPort.Name),
		MaxPeers:          ctx.GlobalUint(cmd.P2PMaxPeers.Name),
		WhitelistCIDR:     ctx.GlobalString(cmd.P2PWhitelist.Name),
		EnableUPnP:        ctx.GlobalBool(cmd.EnableUPnPFlag.Name),
		Encoding:          ctx.GlobalString(cmd.P2PEncoding.Name),
	})
	if err != nil {
		return nil, err
	}
	return svc, nil
}
