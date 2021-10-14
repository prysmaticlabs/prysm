package v2

func WithDataDirectory(dirPath string) Opt {
	return func(node *BeaconNode) {
		node.cfg.dataDir = dirPath
	}
}

func WithBoltMMapInitialSize(size int) Opt {
	return func(node *BeaconNode) {
		node.cfg.mmapInitialSize = size
	}
}

func WithNoDiscovery() Opt {
	return func(node *BeaconNode) {
		node.p2pCfg.NoDiscovery = true
	}
}

func WithStaticPeers(peers []string) Opt {
	return func(node *BeaconNode) {
		node.p2pCfg.StaticPeers = peers
	}
}

func WithRelayNodeAddr(addr string) Opt {
	return func(node *BeaconNode) {
		node.p2pCfg.RelayNodeAddr = addr
	}
}

func WithWeakSubjectivityCheckpoint(checkpoint string) Opt {
	return func(node *BeaconNode) {
		node.cfg.wsCheckpointStr = checkpoint
	}
}
