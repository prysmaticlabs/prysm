package p2p


type BlacklistedPeer struct {
	Peer string
}

type BlacklistedPeers struct {
	Peers []BlacklistedPeer
}


func AddNewBlacklistedPeer(peerID string)  {
	bp := &BlacklistedPeer{Peer: peerID}
	bps := new(BlacklistedPeers)
	bps.Peers = append(bps.Peers, bp)
}


func IsPeerBlacklisted(peerID string) bool {
	bps := new(BlacklistedPeers)
	for i := 0 ; i < len(bps.Peers), i++ {
		if bps.Peers[i].Peer == peerID {
			return true
		}
	}
	return false
}
	
    
}

